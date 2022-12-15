package main

import (
	"sort"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func submitOrder(t *testing.T, ob *OrderBook, o Order) []*Order {
	tr, err := ob.SubmitOrder(&o)
	require.NoError(t, err)

	ol, err := tr.Commit()
	require.NoError(t, err)

	return ol
}

func getQueues(oc *OrderContainer) []*OrderQueue {
	a := make([]*OrderQueue, 0)
	for _, v := range oc.priceTree.Values() {
		a = append(a, v.(*OrderQueue))
	}
	return a
}

func TestLimitOrders(t *testing.T) {
	t.Run("init buy", func(t *testing.T) {
		ob := NewOrderBook()
		expected := []Order{
			{ID: 1, Price: decimal.NewFromFloat(10.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: BuyOrderDirection},
			{ID: 2, Price: decimal.NewFromFloat(10.0), Amount: decimal.NewFromFloat(150.0), Type: LimitOrderType, Dir: BuyOrderDirection},
			{ID: 3, Price: decimal.NewFromFloat(15.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: BuyOrderDirection},
			{ID: 4, Price: decimal.NewFromFloat(15.0), Amount: decimal.NewFromFloat(150.0), Type: LimitOrderType, Dir: BuyOrderDirection},
			{ID: 5, Price: decimal.NewFromFloat(20.0), Amount: decimal.NewFromFloat(150.0), Type: LimitOrderType, Dir: BuyOrderDirection},
			{ID: 6, Price: decimal.NewFromFloat(25.0), Amount: decimal.NewFromFloat(150.0), Type: LimitOrderType, Dir: BuyOrderDirection},
		}
		for _, v := range expected {
			o := submitOrder(t, ob, v)
			require.Equal(t, 0, len(o))
		}

		queues := getQueues(ob.buy)
		sort.SliceStable(queues, func(i, j int) bool {
			return queues[i].Price().LessThan(queues[j].Price())
		})
		require.Equal(t, 4, len(queues))

		{
			e := queues[0].orders.Front()
			require.Equal(t, &expected[0], e.Value.(*Order))
			e = e.Next()
			require.Equal(t, &expected[1], e.Value.(*Order))
			e = e.Next()
			require.Nil(t, e)
		}

		{
			e := queues[1].orders.Front()
			require.Equal(t, &expected[2], e.Value.(*Order))
			e = e.Next()
			require.Equal(t, &expected[3], e.Value.(*Order))
			e = e.Next()
			require.Nil(t, e)
		}

		{
			e := queues[2].orders.Front()
			require.Equal(t, &expected[4], e.Value.(*Order))
			e = e.Next()
			require.Nil(t, e)
		}

		{
			e := queues[3].orders.Front()
			require.Equal(t, &expected[5], e.Value.(*Order))
			e = e.Next()
			require.Nil(t, e)
		}
	})

	t.Run("first buy, part of queue", func(t *testing.T) {
		ob := NewOrderBook()
		sell := []Order{
			{ID: 1, Price: decimal.NewFromFloat(10.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: SellOrderDirection},
			{ID: 2, Price: decimal.NewFromFloat(10.0), Amount: decimal.NewFromFloat(150.0), Type: LimitOrderType, Dir: SellOrderDirection},
			{ID: 3, Price: decimal.NewFromFloat(15.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: SellOrderDirection},
		}
		buy := []Order{
			{ID: 4, Price: decimal.NewFromFloat(15.0), Amount: decimal.NewFromFloat(50.0), Type: LimitOrderType, Dir: BuyOrderDirection},
		}
		for _, v := range sell {
			o := submitOrder(t, ob, v)
			require.Equal(t, 0, len(o))
		}

		{
			o := submitOrder(t, ob, buy[0])
			require.Equal(t, 1, len(o))

			require.Equal(t, o[0], &buy[0])
		}

		queues := getQueues(ob.sell)
		sort.SliceStable(queues, func(i, j int) bool {
			return queues[i].Price().LessThan(queues[j].Price())
		})

		{
			e := queues[0].orders.Front()

			require.Equal(t, &Order{ID: 1, Price: decimal.NewFromFloat(10.0), Amount: decimal.NewFromFloat(50.0), Type: LimitOrderType, Dir: SellOrderDirection}, e.Value.(*Order))
			e = e.Next()
			require.Equal(t, &sell[1], e.Value.(*Order))
			e = e.Next()
			require.Nil(t, e)
		}

		{
			e := queues[1].orders.Front()
			require.Equal(t, &sell[2], e.Value.(*Order))
			e = e.Next()
			require.Nil(t, e)
		}
	})

	t.Run("first buy, full queue", func(t *testing.T) {
		ob := NewOrderBook()
		sell := []Order{
			{ID: 1, Price: decimal.NewFromFloat(10.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: SellOrderDirection},
			{ID: 2, Price: decimal.NewFromFloat(10.0), Amount: decimal.NewFromFloat(150.0), Type: LimitOrderType, Dir: SellOrderDirection},
			{ID: 3, Price: decimal.NewFromFloat(15.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: SellOrderDirection},
		}
		buy := []Order{
			{ID: 4, Price: decimal.NewFromFloat(15.0), Amount: decimal.NewFromFloat(250.0), Type: LimitOrderType, Dir: BuyOrderDirection},
		}
		for _, v := range sell {
			o := submitOrder(t, ob, v)
			require.Equal(t, 0, len(o))
		}

		{
			o := submitOrder(t, ob, buy[0])
			require.Equal(t, 3, len(o))

			require.Equal(t, o[0], &sell[0])
			require.Equal(t, o[1], &sell[1])
			require.Equal(t, o[2], &buy[0])
		}

		queues := getQueues(ob.sell)
		sort.SliceStable(queues, func(i, j int) bool {
			return queues[i].Price().LessThan(queues[j].Price())
		})

		{
			e := queues[0].orders.Front()

			require.Equal(t, &sell[2], e.Value.(*Order))
			e = e.Next()
			require.Nil(t, e)
		}
	})

	t.Run("first buy, all queues", func(t *testing.T) {
		ob := NewOrderBook()
		sell := []Order{
			{ID: 1, Price: decimal.NewFromFloat(10.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: SellOrderDirection},
			{ID: 2, Price: decimal.NewFromFloat(10.0), Amount: decimal.NewFromFloat(150.0), Type: LimitOrderType, Dir: SellOrderDirection},
			{ID: 3, Price: decimal.NewFromFloat(15.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: SellOrderDirection},
		}
		buy := []Order{
			{ID: 4, Price: decimal.NewFromFloat(15.0), Amount: decimal.NewFromFloat(350.0), Type: LimitOrderType, Dir: BuyOrderDirection},
		}
		for _, v := range sell {
			o := submitOrder(t, ob, v)
			require.Equal(t, 0, len(o))
		}

		{
			o := submitOrder(t, ob, buy[0])
			require.Equal(t, 4, len(o))

			require.Equal(t, o[0], &sell[0])
			require.Equal(t, o[1], &sell[1])
			require.Equal(t, o[2], &sell[2])
			require.Equal(t, o[3], &buy[0])
		}

		queues := getQueues(ob.sell)
		sort.SliceStable(queues, func(i, j int) bool {
			return queues[i].Price().LessThan(queues[j].Price())
		})
		require.Equal(t, 0, len(queues))
	})

	t.Run("init sell", func(t *testing.T) {
		ob := NewOrderBook()
		expected := []Order{
			{ID: 1, Price: decimal.NewFromFloat(10.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: SellOrderDirection},
			{ID: 2, Price: decimal.NewFromFloat(10.0), Amount: decimal.NewFromFloat(150.0), Type: LimitOrderType, Dir: SellOrderDirection},
			{ID: 3, Price: decimal.NewFromFloat(15.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: SellOrderDirection},
			{ID: 4, Price: decimal.NewFromFloat(15.0), Amount: decimal.NewFromFloat(150.0), Type: LimitOrderType, Dir: SellOrderDirection},
			{ID: 5, Price: decimal.NewFromFloat(20.0), Amount: decimal.NewFromFloat(150.0), Type: LimitOrderType, Dir: SellOrderDirection},
			{ID: 6, Price: decimal.NewFromFloat(25.0), Amount: decimal.NewFromFloat(150.0), Type: LimitOrderType, Dir: SellOrderDirection},
		}
		for _, v := range expected {
			o := submitOrder(t, ob, v)
			require.Equal(t, 0, len(o))
		}

		queues := getQueues(ob.sell)
		sort.SliceStable(queues, func(i, j int) bool {
			return queues[i].Price().LessThan(queues[j].Price())
		})
		require.Equal(t, 4, len(queues))

		{
			e := queues[0].orders.Front()
			require.Equal(t, &expected[0], e.Value.(*Order))
			e = e.Next()
			require.Equal(t, &expected[1], e.Value.(*Order))
			e = e.Next()
			require.Nil(t, e)
		}

		{
			e := queues[1].orders.Front()
			require.Equal(t, &expected[2], e.Value.(*Order))
			e = e.Next()
			require.Equal(t, &expected[3], e.Value.(*Order))
			e = e.Next()
			require.Nil(t, e)
		}

		{
			e := queues[2].orders.Front()
			require.Equal(t, &expected[4], e.Value.(*Order))
			e = e.Next()
			require.Nil(t, e)
		}

		{
			e := queues[3].orders.Front()
			require.Equal(t, &expected[5], e.Value.(*Order))
			e = e.Next()
			require.Nil(t, e)
		}
	})

	t.Run("first sell, part of queue", func(t *testing.T) {
		ob := NewOrderBook()
		buy := []Order{
			{ID: 1, Price: decimal.NewFromFloat(20.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: BuyOrderDirection},
			{ID: 2, Price: decimal.NewFromFloat(20.0), Amount: decimal.NewFromFloat(150.0), Type: LimitOrderType, Dir: BuyOrderDirection},
			{ID: 3, Price: decimal.NewFromFloat(25.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: BuyOrderDirection},
		}
		sell := []Order{
			{ID: 4, Price: decimal.NewFromFloat(15.0), Amount: decimal.NewFromFloat(50.0), Type: LimitOrderType, Dir: SellOrderDirection},
		}
		for _, v := range buy {
			o := submitOrder(t, ob, v)
			require.Equal(t, 0, len(o))
		}

		{
			o := submitOrder(t, ob, sell[0])
			require.Equal(t, 1, len(o))

			require.Equal(t, o[0], &sell[0])
		}

		queues := getQueues(ob.buy)
		sort.SliceStable(queues, func(i, j int) bool {
			return queues[i].Price().LessThan(queues[j].Price())
		})

		{
			e := queues[0].orders.Front()

			require.Equal(t, &buy[0], e.Value.(*Order))
			e = e.Next()
			require.Equal(t, &buy[1], e.Value.(*Order))
			e = e.Next()
			require.Nil(t, e)
		}

		{
			e := queues[1].orders.Front()
			require.Equal(t, &Order{ID: 3, Price: decimal.NewFromFloat(25.0), Amount: decimal.NewFromFloat(50.0), Type: LimitOrderType, Dir: BuyOrderDirection}, e.Value.(*Order))
			e = e.Next()
			require.Nil(t, e)
		}
	})

	t.Run("first sell, full queue", func(t *testing.T) {
		ob := NewOrderBook()
		buy := []Order{
			{ID: 1, Price: decimal.NewFromFloat(20.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: BuyOrderDirection},
			{ID: 2, Price: decimal.NewFromFloat(20.0), Amount: decimal.NewFromFloat(150.0), Type: LimitOrderType, Dir: BuyOrderDirection},
			{ID: 3, Price: decimal.NewFromFloat(25.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: BuyOrderDirection},
		}
		sell := []Order{
			{ID: 4, Price: decimal.NewFromFloat(15.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: SellOrderDirection},
		}
		for _, v := range buy {
			o := submitOrder(t, ob, v)
			require.Equal(t, 0, len(o))
		}

		{
			o := submitOrder(t, ob, sell[0])
			require.Equal(t, 2, len(o))

			require.Equal(t, o[0], &buy[2])
			require.Equal(t, o[1], &sell[0])
		}

		queues := getQueues(ob.buy)
		sort.SliceStable(queues, func(i, j int) bool {
			return queues[i].Price().LessThan(queues[j].Price())
		})

		{
			e := queues[0].orders.Front()

			require.Equal(t, &buy[0], e.Value.(*Order))
			e = e.Next()
			require.Equal(t, &buy[1], e.Value.(*Order))
			e = e.Next()
			require.Nil(t, e)
		}
	})

	t.Run("first buy, all queues", func(t *testing.T) {
		ob := NewOrderBook()
		buy := []Order{
			{ID: 1, Price: decimal.NewFromFloat(20.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: BuyOrderDirection},
			{ID: 2, Price: decimal.NewFromFloat(20.0), Amount: decimal.NewFromFloat(150.0), Type: LimitOrderType, Dir: BuyOrderDirection},
			{ID: 3, Price: decimal.NewFromFloat(25.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: BuyOrderDirection},
		}
		sell := []Order{
			{ID: 4, Price: decimal.NewFromFloat(15.0), Amount: decimal.NewFromFloat(350.0), Type: LimitOrderType, Dir: SellOrderDirection},
		}
		for _, v := range buy {
			o := submitOrder(t, ob, v)
			require.Equal(t, 0, len(o))
		}

		{
			o := submitOrder(t, ob, sell[0])
			require.Equal(t, 4, len(o))

			require.Equal(t, o[0], &buy[2])
			require.Equal(t, o[1], &buy[0])
			require.Equal(t, o[2], &buy[1])
			require.Equal(t, o[3], &sell[0])
		}

		queues := getQueues(ob.buy)
		sort.SliceStable(queues, func(i, j int) bool {
			return queues[i].Price().LessThan(queues[j].Price())
		})
		require.Equal(t, 0, len(queues))
	})
}

func TestMarketOrders(t *testing.T) {
	t.Run("first buy, part of queue", func(t *testing.T) {
		ob := NewOrderBook()
		sell := []Order{
			{ID: 1, Price: decimal.NewFromFloat(10.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: SellOrderDirection},
			{ID: 2, Price: decimal.NewFromFloat(10.0), Amount: decimal.NewFromFloat(150.0), Type: LimitOrderType, Dir: SellOrderDirection},
			{ID: 3, Price: decimal.NewFromFloat(15.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: SellOrderDirection},
		}
		buy := []Order{
			{ID: 4, Price: decimal.NewFromFloat(15.0), Amount: decimal.NewFromFloat(50.0), Type: MarketOrderType, Dir: BuyOrderDirection},
		}
		for _, v := range sell {
			o := submitOrder(t, ob, v)
			require.Equal(t, 0, len(o))
		}

		{
			o := submitOrder(t, ob, buy[0])
			require.Equal(t, 1, len(o))

			require.Equal(t, o[0], &buy[0])
		}

		queues := getQueues(ob.sell)
		sort.SliceStable(queues, func(i, j int) bool {
			return queues[i].Price().LessThan(queues[j].Price())
		})

		{
			e := queues[0].orders.Front()

			require.Equal(t, &Order{ID: 1, Price: decimal.NewFromFloat(10.0), Amount: decimal.NewFromFloat(50.0), Type: LimitOrderType, Dir: SellOrderDirection}, e.Value.(*Order))
			e = e.Next()
			require.Equal(t, &sell[1], e.Value.(*Order))
			e = e.Next()
			require.Nil(t, e)
		}

		{
			e := queues[1].orders.Front()
			require.Equal(t, &sell[2], e.Value.(*Order))
			e = e.Next()
			require.Nil(t, e)
		}
	})

	t.Run("first buy, full queue", func(t *testing.T) {
		ob := NewOrderBook()
		sell := []Order{
			{ID: 1, Price: decimal.NewFromFloat(10.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: SellOrderDirection},
			{ID: 2, Price: decimal.NewFromFloat(10.0), Amount: decimal.NewFromFloat(150.0), Type: LimitOrderType, Dir: SellOrderDirection},
			{ID: 3, Price: decimal.NewFromFloat(15.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: SellOrderDirection},
		}
		buy := []Order{
			{ID: 4, Price: decimal.NewFromFloat(15.0), Amount: decimal.NewFromFloat(250.0), Type: MarketOrderType, Dir: BuyOrderDirection},
		}
		for _, v := range sell {
			o := submitOrder(t, ob, v)
			require.Equal(t, 0, len(o))
		}

		{
			o := submitOrder(t, ob, buy[0])
			require.Equal(t, 3, len(o))

			require.Equal(t, o[0], &sell[0])
			require.Equal(t, o[1], &sell[1])
			require.Equal(t, o[2], &buy[0])
		}

		queues := getQueues(ob.sell)
		sort.SliceStable(queues, func(i, j int) bool {
			return queues[i].Price().LessThan(queues[j].Price())
		})

		{
			e := queues[0].orders.Front()

			require.Equal(t, &sell[2], e.Value.(*Order))
			e = e.Next()
			require.Nil(t, e)
		}
	})

	t.Run("first buy, all queues", func(t *testing.T) {
		ob := NewOrderBook()
		sell := []Order{
			{ID: 1, Price: decimal.NewFromFloat(10.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: SellOrderDirection},
			{ID: 2, Price: decimal.NewFromFloat(10.0), Amount: decimal.NewFromFloat(150.0), Type: LimitOrderType, Dir: SellOrderDirection},
			{ID: 3, Price: decimal.NewFromFloat(15.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: SellOrderDirection},
		}
		buy := []Order{
			{ID: 4, Price: decimal.NewFromFloat(15.0), Amount: decimal.NewFromFloat(350.0), Type: MarketOrderType, Dir: BuyOrderDirection},
		}
		for _, v := range sell {
			o := submitOrder(t, ob, v)
			require.Equal(t, 0, len(o))
		}

		{
			o := submitOrder(t, ob, buy[0])
			require.Equal(t, 4, len(o))

			require.Equal(t, o[0], &sell[0])
			require.Equal(t, o[1], &sell[1])
			require.Equal(t, o[2], &sell[2])
			require.Equal(t, o[3], &buy[0])
		}

		queues := getQueues(ob.sell)
		sort.SliceStable(queues, func(i, j int) bool {
			return queues[i].Price().LessThan(queues[j].Price())
		})
		require.Equal(t, 0, len(queues))
	})

	t.Run("first sell, part of queue", func(t *testing.T) {
		ob := NewOrderBook()
		buy := []Order{
			{ID: 1, Price: decimal.NewFromFloat(20.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: BuyOrderDirection},
			{ID: 2, Price: decimal.NewFromFloat(20.0), Amount: decimal.NewFromFloat(150.0), Type: LimitOrderType, Dir: BuyOrderDirection},
			{ID: 3, Price: decimal.NewFromFloat(25.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: BuyOrderDirection},
		}
		sell := []Order{
			{ID: 4, Price: decimal.NewFromFloat(15.0), Amount: decimal.NewFromFloat(50.0), Type: MarketOrderType, Dir: SellOrderDirection},
		}
		for _, v := range buy {
			o := submitOrder(t, ob, v)
			require.Equal(t, 0, len(o))
		}

		{
			o := submitOrder(t, ob, sell[0])
			require.Equal(t, 1, len(o))

			require.Equal(t, o[0], &sell[0])
		}

		queues := getQueues(ob.buy)
		sort.SliceStable(queues, func(i, j int) bool {
			return queues[i].Price().LessThan(queues[j].Price())
		})

		{
			e := queues[0].orders.Front()

			require.Equal(t, &buy[0], e.Value.(*Order))
			e = e.Next()
			require.Equal(t, &buy[1], e.Value.(*Order))
			e = e.Next()
			require.Nil(t, e)
		}

		{
			e := queues[1].orders.Front()
			require.Equal(t, &Order{ID: 3, Price: decimal.NewFromFloat(25.0), Amount: decimal.NewFromFloat(50.0), Type: LimitOrderType, Dir: BuyOrderDirection}, e.Value.(*Order))
			e = e.Next()
			require.Nil(t, e)
		}
	})

	t.Run("first sell, full queue", func(t *testing.T) {
		ob := NewOrderBook()
		buy := []Order{
			{ID: 1, Price: decimal.NewFromFloat(20.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: BuyOrderDirection},
			{ID: 2, Price: decimal.NewFromFloat(20.0), Amount: decimal.NewFromFloat(150.0), Type: LimitOrderType, Dir: BuyOrderDirection},
			{ID: 3, Price: decimal.NewFromFloat(25.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: BuyOrderDirection},
		}
		sell := []Order{
			{ID: 4, Price: decimal.NewFromFloat(15.0), Amount: decimal.NewFromFloat(100.0), Type: MarketOrderType, Dir: SellOrderDirection},
		}
		for _, v := range buy {
			o := submitOrder(t, ob, v)
			require.Equal(t, 0, len(o))
		}

		{
			o := submitOrder(t, ob, sell[0])
			require.Equal(t, 2, len(o))

			require.Equal(t, o[0], &buy[2])
			require.Equal(t, o[1], &sell[0])
		}

		queues := getQueues(ob.buy)
		sort.SliceStable(queues, func(i, j int) bool {
			return queues[i].Price().LessThan(queues[j].Price())
		})

		{
			e := queues[0].orders.Front()

			require.Equal(t, &buy[0], e.Value.(*Order))
			e = e.Next()
			require.Equal(t, &buy[1], e.Value.(*Order))
			e = e.Next()
			require.Nil(t, e)
		}
	})

	t.Run("first buy, all queues", func(t *testing.T) {
		ob := NewOrderBook()
		buy := []Order{
			{ID: 1, Price: decimal.NewFromFloat(20.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: BuyOrderDirection},
			{ID: 2, Price: decimal.NewFromFloat(20.0), Amount: decimal.NewFromFloat(150.0), Type: LimitOrderType, Dir: BuyOrderDirection},
			{ID: 3, Price: decimal.NewFromFloat(25.0), Amount: decimal.NewFromFloat(100.0), Type: LimitOrderType, Dir: BuyOrderDirection},
		}
		sell := []Order{
			{ID: 4, Price: decimal.NewFromFloat(15.0), Amount: decimal.NewFromFloat(350.0), Type: MarketOrderType, Dir: SellOrderDirection},
		}
		for _, v := range buy {
			o := submitOrder(t, ob, v)
			require.Equal(t, 0, len(o))
		}

		{
			o := submitOrder(t, ob, sell[0])
			require.Equal(t, 4, len(o))

			require.Equal(t, o[0], &buy[2])
			require.Equal(t, o[1], &buy[0])
			require.Equal(t, o[2], &buy[1])
			require.Equal(t, o[3], &sell[0])
		}

		queues := getQueues(ob.buy)
		sort.SliceStable(queues, func(i, j int) bool {
			return queues[i].Price().LessThan(queues[j].Price())
		})
		require.Equal(t, 0, len(queues))
	})
}
