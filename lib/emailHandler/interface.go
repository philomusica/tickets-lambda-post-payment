package emailHandler

import (
	"github.com/philomusica/tickets-lambda-get-concerts/lib/databaseHandler"
	"github.com/philomusica/tickets-lambda-basket-service/lib/paymentHandler"
)

type EmailHandler interface {
	GenerateTicketPDF(order paymentHandler.Order, concert databaseHandler.Concert, includeQRCode bool, redeemTicketURL string) (attachment []byte)
	SendEmail(order paymentHandler.Order, attachment []byte) (err error)
}
