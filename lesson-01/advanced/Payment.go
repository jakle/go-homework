package main

import (
	"fmt"
	"time"
)

// Payment 支付接口
type Payment interface {
	// Pay 执行支付操作，返回支付结果和错误信息
	Pay(amount float64) (string, error)
	// GetName 获取支付方式名称
	GetName() string
}

// Alipay 支付宝支付
type Alipay struct {
	account string
}

// NewAlipay 创建支付宝支付实例
func NewAlipay(account string) *Alipay {
	return &Alipay{account: account}
}

// Pay 执行支付宝支付操作
func (ali *Alipay) Pay(amount float64) (string, error) {
	time.Sleep(100 * time.Millisecond) // sleep 100毫秒 模拟支付处理
	return fmt.Sprintf("支付宝支付成功: 账户:%s, 金额:%.2f元", ali.account, amount), nil
}

// GetName 获取支付方式名称
func (ali *Alipay) GetName() string {
	return "支付宝"
}

// WechatPay 微信支付
type WechatPay struct {
	openID string
}

// NewWechatPay 创建微信支付实例
func NewWechatPay(openID string) *WechatPay {
	return &WechatPay{openID: openID}
}

// Pay 执行微信支付操作
func (wechat *WechatPay) Pay(amount float64) (string, error) {
	time.Sleep(100 * time.Millisecond) // sleep 100毫秒 模拟支付处理
	return fmt.Sprintf("微信支付成功: OpenID:%s, 金额:%.2f元", wechat.openID, amount), nil
}

// GetName 获取支付方式名称
func (w *WechatPay) GetName() string {
	return "微信支付"
}

// BankCard 银行卡支付
type BankCardPay struct {
	cardNumber string
	bankName   string
}

// NewBankCard 创建银行卡支付实例
func NewBankCard(cardNumber, bankName string) *BankCardPay {
	return &BankCardPay{cardNumber: cardNumber, bankName: bankName}
}

// Pay 执行银行卡支付操作
func (bc *BankCardPay) Pay(amount float64) (string, error) {
	time.Sleep(100 * time.Millisecond)
	return fmt.Sprintf("银行卡支付成功: %s卡号:%s, 金额:%.2f元",
		bc.bankName, bc.cardNumber, amount), nil
}

// GetName 获取支付方式名称
func (bc *BankCardPay) GetName() string {
	return bc.bankName + "银行卡"
}

// PaymentProcess 支付处理器
type PaymentProcess struct {
	payments []Payment
}

// NewPaymentProcess 创建支付处理器实例
func NewPaymentProcess() *PaymentProcess {
	return &PaymentProcess{
		payments: []Payment{},
		//payments: make([]Payment, 0),
		//payments: make([]Payment, 0, 0),

	}
}

// AddPayment 添加支付方式到支付处理器
func (p *PaymentProcess) AddPayment(payment Payment) {
	p.payments = append(p.payments, payment)
}

// ProcessPayment 使用指定索引的支付方式处理支付
func (p *PaymentProcess) ProcessPayment(index int, amount float64) {
	if index < 0 || index >= len(p.payments) { // 判断支付方式是否有效
		fmt.Printf("无效的支付方式: %d\n", index)
		return
	}

	payment := p.payments[index]       // 获取支付方式
	result, err := payment.Pay(amount) // 执行支付
	if err != nil {
		fmt.Printf("%s支付失败: %v\n", payment.GetName(), err)
		return
	}
	fmt.Println(result)
}

func main() {
	fmt.Println("=== 支付系统demo ===")

	var process = NewPaymentProcess()
	process.AddPayment(NewAlipay("1111111@alipay.com"))
	process.AddPayment(NewWechatPay("openid_123456"))
	process.AddPayment(NewBankCard("62134456885454", "招商银行"))

	// 使用不同的支付方式
	amounts := []float64{10.30, 140.00, 50.00}
	for i := 0; i < len(process.payments); i++ {
		process.ProcessPayment(i, amounts[i])
	}
}
