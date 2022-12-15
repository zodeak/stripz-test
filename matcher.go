package main

import (
	"container/list"
	"errors"
	"fmt"

	rbtree "github.com/emirpasic/gods/trees/redblacktree"
	"github.com/shopspring/decimal"
)

type MillisecondTimestamp int64

type OrderID uint64
type OrderType uint8

const (
	MarketOrderType OrderType = iota
	LimitOrderType
)

type OrderDirection uint8

const (
	BuyOrderDirection OrderDirection = iota
	SellOrderDirection
)

type Order struct {
	Amount decimal.Decimal `json:"amount"`
	Price  decimal.Decimal `json:"price"`
	//Timestamp MillisecondTimestamp `json:"timestamp"`
	ID   OrderID        `json:"id"`
	Type OrderType      `json:"type"`
	Dir  OrderDirection `json:"dir"`
}

type priceKey = string
type finalizerFn func()

type OrderContainer struct {
	priceTree *rbtree.Tree // [Order.Price]*OrderQueue
	priceHash map[priceKey]*OrderQueue
	volume    decimal.Decimal
}

func newOrderContainer() *OrderContainer {
	const defaultMapSize = 1024 * 1024

	return &OrderContainer{
		priceTree: rbtree.NewWith(func(a, b any) int {
			return a.(decimal.Decimal).Cmp(b.(decimal.Decimal))
		}),
		priceHash: make(map[priceKey]*OrderQueue, defaultMapSize),
		volume:    decimal.Zero,
	}
}

func (oc *OrderContainer) Debug() {
	fmt.Println(oc.priceTree.String())
	for _, v := range oc.priceTree.Values() {
		q := v.(*OrderQueue)
		q.Debug()
	}
}

func (oc *OrderContainer) Volume() decimal.Decimal {
	return oc.volume
}

func (oc *OrderContainer) Add(order *Order) error {
	priceKey := order.Price.String()
	queue, ok := oc.priceHash[priceKey]
	if !ok {
		queue = newOrderQueue(order.Price)
		oc.priceHash[priceKey] = queue
		oc.priceTree.Put(order.Price, queue)
	}

	queue.Add(order)
	oc.volume.Add(order.Amount)

	return nil
}

func (oc *OrderContainer) Remove(price decimal.Decimal) error {
	priceKey := price.String()
	queue, ok := oc.priceHash[priceKey]
	if !ok {
		return nil
	}
	delete(oc.priceHash, priceKey)

	oc.priceTree.Remove(price)
	oc.volume = oc.volume.Sub(queue.Volume())

	return nil
}

func nextMinNode(cur *rbtree.Node) *rbtree.Node {
	if right := cur.Right; right != nil {
		n := right.Left
		if n == nil {
			return right
		}

		for {
			if n.Left == nil {
				return n
			}
			n = n.Left
		}
	}
	if par := cur.Parent; par != nil {
		for {
			if cur.Key.(decimal.Decimal).LessThan(par.Key.(decimal.Decimal)) {
				return par
			}
			cur = par
			par = cur.Parent
			if par == nil {
				return nil
			}
		}
	}
	return nil
}

func nextMaxNode(cur *rbtree.Node) *rbtree.Node {
	if left := cur.Left; left != nil {
		n := left.Right
		if n == nil {
			return left
		}

		for {
			if n.Right == nil {
				return n
			}
			n = n.Right
		}
	}
	if par := cur.Parent; par != nil {
		for {
			if cur.Key.(decimal.Decimal).GreaterThan(par.Key.(decimal.Decimal)) {
				return par
			}
			cur = par
			par = cur.Parent
			if par == nil {
				return nil
			}
		}
	}
	return nil
}

func (oc *OrderContainer) matchMinPrice(order *Order, stopPrice *decimal.Decimal) ([]*Order, decimal.Decimal, finalizerFn) {
	orders := make([]*Order, 0)
	finalizers := make([]finalizerFn, 0)
	amountLeft := order.Amount

	node := oc.priceTree.Left()
	for node != nil {
		queue := node.Value.(*OrderQueue)
		if stopPrice != nil && queue.Price().GreaterThan(*stopPrice) {
			break
		}

		done, left, finalizer := queue.Process(order, amountLeft)
		if order.Amount.GreaterThanOrEqual(queue.Volume()) {
			// here we can skip queue finalizer, because all of queue orders have been fully processed
			// GC will free it
			finalizers = append(finalizers, func() {
				oc.Remove(queue.Price())
			})
		} else {
			finalizers = append(finalizers, func() {
				finalizer()
			})
		}

		orders = append(orders, done...)

		amountLeft = left
		if left.Equal(decimal.Zero) {
			break
		}
		node = nextMinNode(node)
	}

	return orders, amountLeft, func() {
		for _, fn := range finalizers {
			fn()
		}
	}
}

func (oc *OrderContainer) matchMaxPrice(order *Order, stopPrice *decimal.Decimal) ([]*Order, decimal.Decimal, finalizerFn) {
	orders := make([]*Order, 0)
	finalizers := make([]finalizerFn, 0)
	amountLeft := order.Amount

	node := oc.priceTree.Right()
	for node != nil {
		queue := node.Value.(*OrderQueue)
		if stopPrice != nil && queue.Price().LessThan(*stopPrice) {
			break
		}

		done, left, finalizer := queue.Process(order, amountLeft)
		if order.Amount.GreaterThanOrEqual(queue.Volume()) {
			// here we can skip queue finalizer, because all of queue orders have been fully processed
			// GC will free it
			finalizers = append(finalizers, func() {
				oc.Remove(queue.Price())
			})
		} else {
			finalizers = append(finalizers, func() {
				finalizer()
			})
		}

		orders = append(orders, done...)

		amountLeft = left
		if left.Equal(decimal.Zero) {
			break
		}
		node = nextMaxNode(node)
	}

	return orders, amountLeft, func() {
		for _, fn := range finalizers {
			fn()
		}
	}
}

type OrderQueue struct {
	orders *list.List
	price  decimal.Decimal
	volume decimal.Decimal
}

func newOrderQueue(price decimal.Decimal) *OrderQueue {
	return &OrderQueue{
		orders: list.New(),
		price:  price,
		volume: decimal.Zero,
	}
}

func (oq *OrderQueue) Debug() {
	if oq.orders.Len() > 0 {
		el := oq.orders.Front()
		for el != nil {
			el = el.Next()
		}
	}
}

func (oq *OrderQueue) Price() decimal.Decimal {
	return oq.price
}

func (oq *OrderQueue) Volume() decimal.Decimal {
	return oq.volume
}

func (oq *OrderQueue) Add(order *Order) *list.Element {
	el := oq.orders.PushBack(order)
	oq.volume = oq.volume.Add(order.Amount)
	return el
}

func (oq *OrderQueue) Remove(el *list.Element) {
	order := oq.orders.Remove(el).(*Order)
	oq.volume = oq.volume.Sub(order.Amount)
}

func (oq *OrderQueue) update(order *Order, amount decimal.Decimal) {
	order.Amount = amount
	oq.volume = oq.volume.Sub(amount)
}

func (oq *OrderQueue) Process(order *Order, amount decimal.Decimal) ([]*Order, decimal.Decimal, finalizerFn) {
	if oq.orders.Len() == 0 {
		return nil, decimal.Zero, func() {}
	}

	devastated := make([]*Order, 0)
	finalizers := make([]finalizerFn, 0)

	amountLeft := amount
	el := oq.orders.Front()

	for el != nil {
		currOrder := el.Value.(*Order)
		if amountLeft.LessThan(currOrder.Amount) {
			amount := currOrder.Amount.Sub(amountLeft)
			finalizers = append(finalizers, func() {
				oq.update(currOrder, amount)
			})
			amountLeft = decimal.Zero
			break
		}

		devastated = append(devastated, currOrder)
		finalizers = append(finalizers, func() {
			oq.Remove(el)
		})
		amountLeft = amountLeft.Sub(currOrder.Amount)
		el = el.Next()
	}

	return devastated, amountLeft, func() {
		for _, fn := range finalizers {
			fn()
		}
	}
}

type OrderBook struct {
	buy  *OrderContainer
	sell *OrderContainer
}

func NewOrderBook() *OrderBook {
	return &OrderBook{
		buy:  newOrderContainer(),
		sell: newOrderContainer(),
	}
}

func (ob *OrderBook) Debug() {
	fmt.Println("[buy]")
	ob.buy.Debug()
	fmt.Println("[sell]")
	ob.sell.Debug()
}

func (ob *OrderBook) SubmitOrder(order *Order) (Transaction, error) {
	if order.Price.Sign() <= 0 {
		return Transaction{}, ErrBadPrice
	}
	if order.Amount.Sign() <= 0 {
		return Transaction{}, ErrBadAmount
	}

	if order.Type == MarketOrderType {
		return ob.matchMarketOrder(order)
	}

	return ob.matchLimitOrder(order)
}

// market orders should be processed immediately
func (ob *OrderBook) matchMarketOrder(order *Order) (Transaction, error) {
	if order.Dir == BuyOrderDirection {
		if order.Amount.LessThan(ob.sell.Volume()) {
			return newTransaction(nil, func() {}), nil
		}

		doneOrders, amountLeft, finalizer := ob.sell.matchMinPrice(order, &order.Price)
		if amountLeft.GreaterThan(decimal.Zero) {
			panic("market volume assert")
		}
		doneOrders = append(doneOrders, order)
		return newTransaction(doneOrders, finalizer), nil
	}

	if order.Amount.LessThan(ob.buy.Volume()) {
		return newTransaction(nil, func() {}), nil
	}

	doneOrders, amountLeft, finalizer := ob.buy.matchMaxPrice(order, &order.Price)
	if amountLeft.GreaterThan(decimal.Zero) {
		panic("market volume assert")
	}
	doneOrders = append(doneOrders, order)
	return newTransaction(doneOrders, finalizer), nil
}

func (ob *OrderBook) matchLimitOrder(order *Order) (Transaction, error) {
	if order.Dir == BuyOrderDirection {
		doneOrders, amountLeft, finalizer := ob.sell.matchMinPrice(order, &order.Price)
		if amountLeft.GreaterThan(decimal.Zero) {
			return newTransaction(doneOrders, func() {
				finalizer()
				order.Amount = amountLeft
				ob.buy.Add(order)
			}), nil
		}
		doneOrders = append(doneOrders, order)
		return newTransaction(doneOrders, finalizer), nil
	}

	doneOrders, amountLeft, finalizer := ob.buy.matchMaxPrice(order, &order.Price)
	if amountLeft.GreaterThan(decimal.Zero) {
		return newTransaction(doneOrders, func() {
			finalizer()
			order.Amount = amountLeft
			ob.sell.Add(order)
		}), nil
	}
	doneOrders = append(doneOrders, order)
	return newTransaction(doneOrders, finalizer), nil
}

type Transaction struct {
	orders   []*Order
	finalize finalizerFn
}

func newTransaction(orders []*Order, finalize finalizerFn) Transaction {
	return Transaction{
		orders:   orders,
		finalize: finalize,
	}
}

func (tr *Transaction) Commit() ([]*Order, error) {
	if tr.finalize != nil {
		tr.finalize()
		tr.finalize = nil
	}
	return tr.orders, nil
}

func (tr *Transaction) Rollback() error {
	tr.orders = nil
	tr.finalize = nil
	return nil
}

var (
	ErrBadPrice  = errors.New("bad price value")
	ErrBadAmount = errors.New("bad amount value")
)
