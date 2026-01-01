package main

import (
	"errors"
	"fmt"
)

// 自定义错误
var (
	ErrorAccountNotFound     = errors.New("账户不存在")    // 账户不存在错误
	ErrorInsufficientBalance = errors.New("余额不足")      // 余额不足错误
	ErrorInvalidAmount       = errors.New("金额必须大于0") // 无效金额错误
)

// Account 银行账户
type Account struct {
	AccountNumber string  // 账户号码
	AccountHolder string  // 账户持有人姓名
	Balance       float64 // 账户余额
	IsActive      bool    // 账户是否激活（未冻结）
}

// Bank 银行系统
type Bank struct {
	accounts map[string]*Account
}

// 创建银行系统
func NewBank() *Bank {
	return &Bank{
		accounts: make(map[string]*Account), // 初始化账户映射表
	}
}

// OpenAccount 开户方法，参数为账户号码、账户持有人姓名和初始存款金额
func (b *Bank) OpenAccount(accountNumber string, accountHolder string, initialAccount float64) error {
	if initialAccount < 0 {
		return ErrorInvalidAmount
	}

	if _, exists := b.accounts[accountNumber]; exists {
		return fmt.Errorf("账户 %s 已存在", accountNumber)
	}

	// 创建新账户并添加到银行系统中
	b.accounts[accountNumber] = &Account{
		AccountNumber: accountNumber,  // 账户号码
		AccountHolder: accountHolder,  // 账户持有人
		Balance:       initialAccount, // 初始余额
		IsActive:      true,           // 新账户默认激活
	}
	return nil
}

/**
** Deposit 存款方法
** accountNumber 账户号码
** amount 存款金额
 */
func (b *Bank) Deposit(accountNumber string, amount float64) error {
	if amount <= 0 {
		return ErrorInvalidAmount
	}

	account, exists := b.accounts[accountNumber]
	if !exists || !account.IsActive {
		return ErrorAccountNotFound
	}

	account.Balance += amount // 增加账户余额
	return nil
}

/**
** Withdraw 取款方法
** accountNumber 账户号码
** amount 取款金额
 */
func (b *Bank) Withdraw(accountNumber string, amount float64) error {
	if amount <= 0 {
		return ErrorInvalidAmount
	}

	account, exists := b.accounts[accountNumber]
	if !exists || !account.IsActive {
		return ErrorAccountNotFound
	}

	if account.Balance < amount {
		return ErrorInsufficientBalance
	}

	account.Balance -= amount // 减少账户余额
	return nil
}

/**
** GetBalance 查询余额方法
** accountNumber 账户号码
** @return 余额和错误信息
 */
func (b *Bank) GetBalance(accountNumber string) (float64, error) {
	account, exists := b.accounts[accountNumber]
	if !exists || !account.IsActive {
		return 0, ErrorAccountNotFound
	}
	return account.Balance, nil
}

/**
** Transfer 转账方法
** fromAccount 转出账户
** toAccount 转入账户
** amount 转入账户和转账金额
 */
func (b *Bank) Transfer(fromAccount, toAccount string, amount float64) error {
	if amount <= 0 {
		return ErrorInvalidAmount
	}

	// 检查源账户
	fromAcc, exists := b.accounts[fromAccount]
	if !exists || !fromAcc.IsActive {
		return fmt.Errorf("源账户 %s 不存在或已冻结", fromAccount)
	}

	// 检查目标账户
	toAcc, exists := b.accounts[toAccount]
	if !exists || !toAcc.IsActive {
		return fmt.Errorf("目标账户 %s 不存在或已冻结", toAccount)
	}

	// 检查余额
	if fromAcc.Balance < amount {
		return ErrorInsufficientBalance
	}

	// 执行转账操作
	fromAcc.Balance -= amount // 源账户余额减少
	toAcc.Balance += amount   // 目标账户余额增加

	return nil
}

/**
** FreezeAccount 冻结账户方法
** accountNumber 账户号码
 */
func (b *Bank) FreezeAccount(accountNumber string) error {
	account, exists := b.accounts[accountNumber]
	if !exists {
		return ErrorAccountNotFound
	}
	account.IsActive = false // 设置账户为非激活状态（冻结）
	return nil
}

/**
** UnfreezeAccount 解冻账户方法
** accountNumber 账户号码
 */
func (b *Bank) UnfreezeAccount(accountNumber string) error {
	account, exists := b.accounts[accountNumber]
	if !exists {
		return ErrorAccountNotFound
	}
	account.IsActive = true // 设置账户为激活状态（解冻）
	return nil
}

/**
**显示所有账户信息
 */
func (b *Bank) DisplayAllAccounts() {
	fmt.Println("\n=== 账户列表 ===")
	if len(b.accounts) == 0 {
		fmt.Println("暂无账户")
		return
	}

	totalBalance := 0.0 // 银行总存款余额
	for _, account := range b.accounts {
		status := "正常"
		if !account.IsActive {
			status = "冻结"
		}
		fmt.Printf("账号: %s, 户主: %s, 余额: ¥%.2f, 状态: %s\n",
			account.AccountNumber, account.AccountHolder, account.Balance, status)
		totalBalance += account.Balance
	}
	fmt.Printf("总余额: ¥%.2f\n", totalBalance)
}

func main() {
	bank := NewBank()

	// 开户信息列表，包含账户号码、持有人和初始存款
	bank.OpenAccount("1", "张三", 1700.0)
	bank.OpenAccount("2", "李四", 600.0)
	bank.OpenAccount("3", "王五", 14000.0)

	// 显示所有账户
	bank.DisplayAllAccounts()

	// 存款操作
	if err := bank.Deposit("1", 500.0); err == nil {
		fmt.Println("存款成功，账号1存款%f", 500.0)
	}

	// 转账操作
	if err := bank.Transfer("1", "3", 300.0); err == nil {
		fmt.Println("转账成功,账号1向账号3转账 %f", 300.0)
	}

	// 取款操作
	if err := bank.Withdraw("2", 200.0); err == nil {
		fmt.Println("取款成功，账号2取款%f", 200.0)
	}

	// 尝试超额取款（测试错误处理）
	if err := bank.Withdraw("2", 1000.0); err != nil {
		fmt.Printf("取款失败: %v\n", err)
	}

	// 查询余额
	if balance, err := bank.GetBalance("1"); err == nil {
		fmt.Printf("账户1余额: ¥%.2f\n", balance)
	}

	// 冻结账户
	if err := bank.FreezeAccount("3"); err == nil {
		fmt.Println("账户3已冻结")
	}

	// 尝试向冻结账户转账
	if err := bank.Transfer("1", "3", 100.0); err != nil {
		fmt.Printf("转账失败: %v\n", err)
	}

	bank.DisplayAllAccounts()

	// 解冻账户
	if err := bank.UnfreezeAccount("3"); err == nil {
		fmt.Println("账户3已解冻")
	}

	bank.DisplayAllAccounts()

}
