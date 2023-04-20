package emailHandler

import (
	"github.com/philomusica/tickets-lambda-basket-service/lib/paymentHandler"
	"github.com/philomusica/tickets-lambda-get-concerts/lib/databaseHandler"
)

type EmailHandler interface {
	GenerateTicketPDF(order paymentHandler.Order, concert databaseHandler.Concert, includeQRCode bool, redeemTicketURL string) (attachment []byte)
	SendEmail(order paymentHandler.Order, attachment []byte) (err error)
	SendPaymentFailureEmail(order paymentHandler.Order) (err error)
}
