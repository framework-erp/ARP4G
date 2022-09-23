package test

import (
	"ARP4G/arp"
	"ARP4G/repoimpl"
	"context"
	"runtime"
	"testing"
	"time"
)

//业务开发测试
func TestPlaceOrder(t *testing.T) {
	orderService := &OrderService{
		repoimpl.NewMemRepository(func() *Product { return &Product{} }),
		repoimpl.NewMemRepository(func() *ProductStock { return &ProductStock{} }),
		repoimpl.NewMemRepository(func() *Order { return &Order{} })}

	err := arp.Go(context.Background(), func(ctx context.Context) {
		orderService.NewProduct(ctx, 1, "apple", 10)
	})
	AssertNoError(t, err)

	err = arp.Go(context.Background(), func(ctx context.Context) {
		orderService.IncreaseStock(ctx, 1, 5)
	})
	AssertNoError(t, err)

	err = arp.Go(context.Background(), func(ctx context.Context) {
		orderService.NewProduct(ctx, 2, "orange", 5)
	})
	AssertNoError(t, err)

	err = arp.Go(context.Background(), func(ctx context.Context) {
		orderService.IncreaseStock(ctx, 2, 5)
	})
	AssertNoError(t, err)

	orderItems := make(map[int]int)
	orderItems[1] = 2
	orderItems[2] = 3

	orderId := 1
	userId := 1
	userAddress := "address"
	var bizErr error
	err = arp.Go(context.Background(), func(ctx context.Context) {
		bizErr = orderService.PlaceOrder(ctx, orderId, orderItems, userId, userAddress)
	})
	AssertNoError(t, err)
	AssertNoError(t, bizErr)

	orderId = 2
	err = arp.Go(context.Background(), func(ctx context.Context) {
		bizErr = orderService.PlaceOrder(ctx, orderId, orderItems, userId, userAddress)
	})
	AssertNoError(t, err)
	AssertError(t, bizErr)

	orderItems[1] = 2
	orderItems[2] = 2
	orderId = 3
	err = arp.Go(context.Background(), func(ctx context.Context) {
		bizErr = orderService.PlaceOrder(ctx, orderId, orderItems, userId, userAddress)
	})
	AssertNoError(t, err)
	AssertNoError(t, bizErr)

}

//并发安全测试
func TestModifyStock(t *testing.T) {
	orderService := &OrderService{
		repoimpl.NewMemRepository(func() *Product { return &Product{} }),
		repoimpl.NewMemRepository(func() *ProductStock { return &ProductStock{} }),
		repoimpl.NewMemRepository(func() *Order { return &Order{} })}

	err := arp.Go(context.Background(), func(ctx context.Context) {
		orderService.NewProduct(ctx, 1, "apple", 10)
	})
	AssertNoError(t, err)

	err = arp.Go(context.Background(), func(ctx context.Context) {
		stock := orderService.IncreaseStock(ctx, 1, 1000)
		AssertEqual(t, 1000, stock.freeAmount)
	})
	AssertNoError(t, err)

	go func() {
		arp.Go(context.Background(), func(ctx context.Context) {
			for i := 0; i < 100; i++ {
				orderService.IncreaseStock(ctx, 1, 1)
				runtime.Gosched()
			}
		})
	}()

	go func() {
		arp.Go(context.Background(), func(ctx context.Context) {
			for i := 0; i < 100; i++ {
				orderService.DecreaseStock(ctx, 1, 1)
				runtime.Gosched()
			}
		})
	}()

	go func() {
		arp.Go(context.Background(), func(ctx context.Context) {
			for i := 0; i < 100; i++ {
				orderService.IncreaseStock(ctx, 1, 2)
				runtime.Gosched()
			}
		})
	}()

	go func() {
		arp.Go(context.Background(), func(ctx context.Context) {
			for i := 0; i < 100; i++ {
				orderService.DecreaseStock(ctx, 1, 2)
				runtime.Gosched()
			}
		})
	}()

	time.Sleep(1 * time.Second)
	stock := orderService.FindStock(context.Background(), 1)
	AssertEqual(t, 1000, stock.freeAmount)
}
