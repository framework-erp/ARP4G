package test

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/framework-arp/ARP4G/arp"
	"github.com/framework-arp/ARP4G/repoimpl"
)

//业务开发测试
func TestPlaceOrder(t *testing.T) {
	orderService := &OrderService{
		repoimpl.NewMemRepository(func() *Product { return &Product{} }),
		repoimpl.NewMemRepository(func() *ProductStock { return &ProductStock{} }),
		repoimpl.NewMemRepository(func() *Order { return &Order{} })}

	err := arp.Go(context.Background(), func(ctx context.Context) error {
		orderService.NewProduct(ctx, 1, "apple", 10)
		return nil
	})
	AssertNoError(t, err)

	err = arp.Go(context.Background(), func(ctx context.Context) error {
		orderService.IncreaseStock(ctx, 1, 5)
		return nil
	})
	AssertNoError(t, err)

	err = arp.Go(context.Background(), func(ctx context.Context) error {
		orderService.NewProduct(ctx, 2, "orange", 5)
		return nil
	})
	AssertNoError(t, err)

	err = arp.Go(context.Background(), func(ctx context.Context) error {
		orderService.IncreaseStock(ctx, 2, 5)
		return nil
	})
	AssertNoError(t, err)

	orderItems := make(map[int]int)
	orderItems[1] = 2
	orderItems[2] = 3

	orderId := 1
	userId := 1
	userAddress := "address"
	err = arp.Go(context.Background(), func(ctx context.Context) error {
		return orderService.PlaceOrder(ctx, orderId, orderItems, userId, userAddress)
	})
	AssertNoError(t, err)

	orderId = 2
	err = arp.Go(context.Background(), func(ctx context.Context) error {
		return orderService.PlaceOrder(ctx, orderId, orderItems, userId, userAddress)
	})
	AssertError(t, err)

	orderItems[1] = 2
	orderItems[2] = 2
	orderId = 3
	err = arp.Go(context.Background(), func(ctx context.Context) error {
		return orderService.PlaceOrder(ctx, orderId, orderItems, userId, userAddress)
	})
	AssertNoError(t, err)

}

//并发安全测试
func TestModifyStock(t *testing.T) {
	orderService := &OrderService{
		repoimpl.NewMemRepository(func() *Product { return &Product{} }),
		repoimpl.NewMemRepository(func() *ProductStock { return &ProductStock{} }),
		repoimpl.NewMemRepository(func() *Order { return &Order{} })}

	err := arp.Go(context.Background(), func(ctx context.Context) error {
		orderService.NewProduct(ctx, 1, "apple", 10)
		return nil
	})
	AssertNoError(t, err)

	err = arp.Go(context.Background(), func(ctx context.Context) error {
		stock := orderService.IncreaseStock(ctx, 1, 1000)
		AssertEqual(t, 1000, stock.freeAmount)
		return nil
	})
	AssertNoError(t, err)

	go func() {
		arp.Go(context.Background(), func(ctx context.Context) error {
			for i := 0; i < 100; i++ {
				orderService.IncreaseStock(ctx, 1, 1)
				runtime.Gosched()
			}
			return nil
		})
	}()

	go func() {
		arp.Go(context.Background(), func(ctx context.Context) error {
			for i := 0; i < 100; i++ {
				orderService.DecreaseStock(ctx, 1, 1)
				runtime.Gosched()
			}
			return nil
		})
	}()

	go func() {
		arp.Go(context.Background(), func(ctx context.Context) error {
			for i := 0; i < 100; i++ {
				orderService.IncreaseStock(ctx, 1, 2)
				runtime.Gosched()
			}
			return nil
		})
	}()

	go func() {
		arp.Go(context.Background(), func(ctx context.Context) error {
			for i := 0; i < 100; i++ {
				orderService.DecreaseStock(ctx, 1, 2)
				runtime.Gosched()
			}
			return nil
		})
	}()

	time.Sleep(1 * time.Second)
	stock := orderService.FindStock(context.Background(), 1)
	AssertEqual(t, 1000, stock.freeAmount)
}
