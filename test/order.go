package test

import (
	"context"
	"errors"
)

type Order struct {
	id          int
	items       []OrderItem
	userId      int
	userAddress string
	state       int
}

type OrderItem struct {
	product Product
	amount  int
}

type Product struct {
	id    int
	name  string
	price int
}

type ProductStock struct {
	id         int //product id
	freeAmount int
}

func (stock *ProductStock) Increase(amount int) {
	stock.freeAmount += amount
}

func (stock *ProductStock) Decrease(amount int) {
	stock.freeAmount -= amount
}

func (stock *ProductStock) CheckAmount(amount int) bool {
	return stock.freeAmount >= amount
}

type OrderService struct {
	productRepository      ProductRepository
	productStockRepository ProductStockRepository
	orderRepository        OrderRepository
}

type ProductRepository interface {
	Find(ctx context.Context, id any) *Product
	Put(ctx context.Context, entity *Product)
}

type ProductStockRepository interface {
	Find(ctx context.Context, id any) *ProductStock
	Take(ctx context.Context, id any) *ProductStock
	PutIfAbsent(ctx context.Context, productStock *ProductStock) (actual *ProductStock, absent bool)
	TakeOrPutIfAbsent(ctx context.Context, id any, productStock *ProductStock) *ProductStock
}

type OrderRepository interface {
	Take(ctx context.Context, id any) *Order
	Put(ctx context.Context, order *Order)
}

func (orderService *OrderService) NewProduct(ctx context.Context, id int, name string, price int) {
	orderService.productRepository.Put(ctx, &Product{id, name, price})
}

func (orderService *OrderService) IncreaseStock(ctx context.Context, productId int, amount int) *ProductStock {
	stock := orderService.productStockRepository.TakeOrPutIfAbsent(ctx, productId, &ProductStock{productId, 0})
	stock.Increase(amount)
	return stock
}

func (orderService *OrderService) DecreaseStock(ctx context.Context, productId int, amount int) *ProductStock {
	stock := orderService.productStockRepository.TakeOrPutIfAbsent(ctx, productId, &ProductStock{productId, 0})
	stock.Decrease(amount)
	return stock
}

func (orderService *OrderService) PlaceOrder(ctx context.Context, orderId int, orderItems map[int]int, userId int, userAddress string) error {
	for productId, amount := range orderItems {
		stock := orderService.productStockRepository.Take(ctx, productId)
		if !stock.CheckAmount(amount) {
			return errors.New("insufficient stock")
		}
	}
	items := make([]OrderItem, 1)
	for productId, amount := range orderItems {
		stock := orderService.productStockRepository.Take(ctx, productId)
		stock.Decrease(amount)
		product := orderService.productRepository.Find(ctx, productId)
		items = append(items, OrderItem{*product, amount})
	}
	order := &Order{orderId, items, userId, userAddress, 0}
	orderService.orderRepository.Put(ctx, order)
	return nil
}

func (orderService *OrderService) FindStock(ctx context.Context, productId int) *ProductStock {
	stock := orderService.productStockRepository.Find(ctx, productId)
	return stock
}
