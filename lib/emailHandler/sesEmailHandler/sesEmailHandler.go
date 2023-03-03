package sesEmailHandler

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go/service/sesv2"
	"github.com/aws/aws-sdk-go/service/sesv2/sesv2iface"
	"github.com/philomusica/tickets-lambda-get-concerts/lib/databaseHandler"
	"github.com/philomusica/tickets-lambda-basket-service/lib/paymentHandler"
	"github.com/signintech/gopdf"
	"github.com/skip2/go-qrcode"
	"gopkg.in/gomail.v2"
)

// ===============================================================================================================================
// TYPE DEFINITIONS
// ===============================================================================================================================

type SESEmailHandler struct {
	svc           sesv2iface.SESV2API
	senderAddress string
}

// ===============================================================================================================================
// END TYPE DEFINITIONS
// ===============================================================================================================================

// ===============================================================================================================================
// PRIVATE FUNCTIONS
// ===============================================================================================================================

// addDetailToPDF takes a pointer to a Go.Pdf struct and key and value strings, and writes them to the pdf
func addDetailToPDF(pdf *gopdf.GoPdf, key string, value string) {
	yPos := pdf.GetY() + 30.0
	pdf.SetXY(pdf.MarginLeft(), yPos)
	pdf.SetFont("nunito-light", "", 11)
	pdf.Cell(nil, fmt.Sprintf("%s:", key))
	yPos += 15.0
	pdf.SetXY(pdf.MarginLeft(), yPos)
	pdf.SetFont("nunito", "", 14)
	pdf.Cell(nil, value)
}

// buildAdmitString takes an order struct and returns a formatted string indicating how many people to admit (e.g. "2 adults and 1 concession")
func buildAdmitString(order paymentHandler.Order) string {
	var admitString strings.Builder
	if order.NumOfFullPrice > 0 {
		admitString.WriteString(fmt.Sprintf("%d ", order.NumOfFullPrice))
		var ticketType string
		if order.NumOfFullPrice == 1 {
			ticketType = "adult"
		} else {
			ticketType = "adults"
		}
		admitString.WriteString(ticketType)
	}

	if order.NumOfConcessions > 0 {
		if admitString.Len() > 0 {
			admitString.WriteString(" and ")
		}
		admitString.WriteString(fmt.Sprintf("%d ", order.NumOfConcessions))
		var ticketType string
		if order.NumOfConcessions == 1 {
			ticketType = "concession"
		} else {
			ticketType = "concessions"
		}
		admitString.WriteString(ticketType)
	}
	return admitString.String()
}

// ===============================================================================================================================
// END PRIVATE FUNCTIONS
// ===============================================================================================================================

// ===============================================================================================================================
// PUBLIC FUNCTIONS
// ===============================================================================================================================

// GenerateTicketPDF takes an order struct and returns a PDF file in a byte array and an error if fails
func (s SESEmailHandler) GenerateTicketPDF(order paymentHandler.Order, concert databaseHandler.Concert, includeQRCode bool, redeemTicketURL string) (attachment []byte) {
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	marginSize := 30.0
	pdf.SetMargins(marginSize, marginSize, marginSize, marginSize)

	// Load fonts font
	pdf.AddTTFFont("nunito", "./Nunito-Regular.ttf")
	pdf.AddTTFFont("nunito-bold", "./Nunito-Bold.ttf")
	pdf.AddTTFFont("nunito-light", "./Nunito-Light.ttf")

	// Add header
	pdf.AddHeader(func() {
		pdf.SetFont("nunito", "", 14)
		pdf.Cell(nil, "PHILOMUSICA PRESENTS:")
		pdf.SetXY(marginSize, marginSize+25.0)
		pdf.SetFont("nunito-bold", "", 20)
		pdf.Cell(nil, strings.ToUpper(concert.Title))
	})

	// Add one page
	pdf.AddPage()

	pdf.SetY(pdf.GetY() + 30.0)
	addDetailToPDF(&pdf, "Location", concert.Location)

	if includeQRCode && redeemTicketURL != "" {
		var qrcodeImage []byte
		qrcodeImage, _ = qrcode.Encode(fmt.Sprintf("%s/%s/%s", redeemTicketURL, concert.ID, order.Reference), qrcode.Medium, 360)
		ih, _ := gopdf.ImageHolderByBytes(qrcodeImage)
		pdf.ImageByHolder(ih, gopdf.PageSizeA4.W/2+30.0, pdf.GetY(), nil)
	}

	addDetailToPDF(&pdf, "Date", fmt.Sprintf("%s @ %s", concert.Date, concert.Time))
	addDetailToPDF(&pdf, "Name", fmt.Sprintf("%s %s", order.FirstName, order.LastName))
	addDetailToPDF(&pdf, "Reference", order.Reference)
	addDetailToPDF(&pdf, "Admit", buildAdmitString(order))
	return pdf.GetBytesPdf()
}

// New takes an SES V2 interface and sender email address and returns a newly created SESEmailHandler struct
func New(svc sesv2iface.SESV2API, senderAddress string) SESEmailHandler {
	return SESEmailHandler{
		svc,
		senderAddress,
	}
}

// SendEmail takes an order struct and attachment (in bytes) and sends an email to the customer, using the AWS SES v2 API. Returns an error if fails, or nil if successful
func (s SESEmailHandler) SendEmail(order paymentHandler.Order, attachment []byte) (err error) {
	msg := gomail.NewMessage()
	msg.SetHeader("To", order.Email)
	msg.SetHeader("From", s.senderAddress)
	msg.SetHeader("Subject", "Order Confirmation")
	msg.SetBody("text/html", fmt.Sprintf("Dear %s\n\nMany thanks for purchasing tickets to Philomusica's concert. Your eTicket is attached as a PDF to this email. Please bring this PDF with you to the concert, either in digital or paper form.\nWe look forward to seeing you there!\n\nPhilomusica\n\nPlease consider the enviornment before printing your eTicket", order.FirstName))
	msg.Attach(
		"philomusica-concert-tickets.pdf",
		gomail.SetCopyFunc(func(w io.Writer) error {
			_, err := w.Write(attachment)
			return err
		}),
		gomail.SetHeader(map[string][]string{"Content-Type": {"application/pdf"}}),
	)

	var rawEmail bytes.Buffer
	msg.WriteTo(&rawEmail)

	// send raw email
	_, err = s.svc.SendEmail(
		&sesv2.SendEmailInput{
			Content: &sesv2.EmailContent{
				Raw: &sesv2.RawMessage{Data: rawEmail.Bytes()},
			},
			Destination: &sesv2.Destination{
				ToAddresses: []*string{&order.Email},
			},
			FromEmailAddress: &s.senderAddress,
		},
	)

	return
}

// ===============================================================================================================================
// END PUBLIC FUNCTIONS
// ===============================================================================================================================
