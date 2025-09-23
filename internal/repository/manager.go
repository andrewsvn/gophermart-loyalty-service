package repository

type Manager interface {
	GetUserStorage() UserStorage
	GetLoyaltyStorage() LoyaltyStorage
	Close()
}
