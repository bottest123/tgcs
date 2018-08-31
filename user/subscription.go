package user

type Subscription struct {
	Status      SubscriptionStatus
	ChannelName string
	IsTrading   bool
}

// статус подписок
type SubscriptionStatus string

const (
	NonApproved            SubscriptionStatus = "Подписка неактивна"
	Active                                    = "Подписка активна"
	NotFound                                  = "Данный канал не найден"
	ProcessingSubscription                    = "Канал проходит проверку"
)
