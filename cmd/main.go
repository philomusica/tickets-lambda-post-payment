package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/sesv2"
	"github.com/philomusica/tickets-lambda-basket-service/lib/paymentHandler"
	"github.com/philomusica/tickets-lambda-basket-service/lib/paymentHandler/stripePaymentHandler"
	"github.com/philomusica/tickets-lambda-get-concerts/lib/databaseHandler"
	"github.com/philomusica/tickets-lambda-get-concerts/lib/databaseHandler/ddbHandler"
	"github.com/philomusica/tickets-lambda-post-payment/lib/emailHandler"
	"github.com/philomusica/tickets-lambda-post-payment/lib/emailHandler/sesEmailHandler"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/webhook"
)

// ===============================================================================================================================
// GLOBAL VARIABLES
// ===============================================================================================================================

const (
	PROCESSING string = "processing"
	COMPLETE string = "complete"
	PAYMENT_FAILED string = "payment-failed"
)
// ===============================================================================================================================
// END GLOBAL VARIABLES
// ===============================================================================================================================

// ===============================================================================================================================
// TYPE DEFINITIONS
// ===============================================================================================================================

// ===============================================================================================================================
// END TYPE DEFINITIONS
// ===============================================================================================================================

// ===============================================================================================================================
// PRIVATE FUNCTIONS
// ===============================================================================================================================

// ===============================================================================================================================
// END PRIVATE FUNCTIONS
// ===============================================================================================================================

func processFailedPayment(orders []paymentHandler.Order, dbHandler databaseHandler.DatabaseHandler, emailHandler emailHandler.EmailHandler) (err error) {

	for _, order := range orders {
		fmt.Printf("calling UpdateOrderInTable with %s, %s, %s\n", order.ConcertID, order.OrderReference, PAYMENT_FAILED)
		err = dbHandler.UpdateOrderInTable(order.ConcertID, order.OrderReference, PAYMENT_FAILED)
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	if len(orders) != 0 {
		err = emailHandler.SendPaymentFailureEmail(orders[0])
	}
	return
}

// postPaymentHandler takes the events.APIGatewayProxyRequest struct, the databaseHandler, stripeHandler and sesHandler, parse the JSON object from Stripe, updates the Orders table to with payment status set to "complete", updates the number of tickets sold for the relevant concert(s), generates a PDF eticket and emails the customer with the PDF attached, it returns an events.APIGatewayProxyResponse and error which should be passed on by the lambda Handler function
func postPaymentHandler(request events.APIGatewayProxyRequest, dbHandler databaseHandler.DatabaseHandler, paymentHandler paymentHandler.PaymentHandler, emailHandler emailHandler.EmailHandler) (response events.APIGatewayProxyResponse, err error) {
	fmt.Println("postPaymentHandler called")
	response.StatusCode = 500
	response.Body = "Error processing webhook"

	event := stripe.Event{}

	bodyAsBytes := []byte(request.Body)
	if err = json.Unmarshal(bodyAsBytes, &event); err != nil {
		fmt.Printf("Error unmarshalling json from Stripe: %s\n", err)
		return
	}

	// No need to check for empty string here, as it will already have been checked by Handler (the calling function)
	stripeSecret := os.Getenv("STRIPE_SECRET")

	// Validate the Stripe signature
	signatureHeader := request.Headers["Stripe-Signature"]
	event, err = webhook.ConstructEvent(bodyAsBytes, signatureHeader, stripeSecret)
	if err != nil {
		fmt.Printf("Unable to validate stripe event and header: %s\n", err)
		return
	}

	// Get paymentIntent from Stripe event struct
	var paymentIntent stripe.PaymentIntent
	if err = json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		fmt.Printf("Unable to unmarshal paymentIntent from Stripe's event struct: %T\n", err)
		return
	}

	reference := paymentIntent.Metadata["order_reference"]
	orders, err := dbHandler.GetOrdersByOrderReferenceFromTable(reference)
	if err != nil || len(orders) == 0 {
		fmt.Printf("Unable to get orders by reference id: %T\n", err)
		return
	}

	if event.Type == "payment_intent.payment_failed" {
		err = processFailedPayment(orders, dbHandler, emailHandler)
		if err != nil {
			fmt.Println(err)
		}
		return
	}

	for _, order := range orders {
		// Update order in table to show complete
		if event.Type == "payment_intent.processing" {
			err = dbHandler.UpdateOrderInTable(order.ConcertID, order.OrderReference, PROCESSING)
		} else if event.Type == "payment_intent.succeeded" {
			err = dbHandler.UpdateOrderInTable(order.ConcertID, order.OrderReference, COMPLETE)
		}

		if err != nil {
			fmt.Printf("Error updating order in table: %v\n", err)
			return
		}

		// If status was processing, user will have already received email, etc, so nothing more to be done
		if order.OrderStatus == "processing" {
			continue
		}

		// Update concert table with number of sold tickets
		err = dbHandler.UpdateTicketsSoldInTable(order.ConcertID, uint16(order.NumOfFullPrice+order.NumOfConcessions))
		if err != nil {
			fmt.Println(err)
			return
		}
		// Generate QR code
		redeemTicketURL := os.Getenv("REDEEM_TICKET_API")
		if redeemTicketURL == "" {
			fmt.Printf("redeemTicketURL not set\n")
			return
		}

		var concert *databaseHandler.Concert
		concert, err = dbHandler.GetConcertFromTable(order.ConcertID)
		if err != nil {
			fmt.Printf("Unable to get concert from concerts table: %T\n", err)
			return
		}

		err = dbHandler.ReformatDateTimeAndTickets(concert)
		if err != nil {
			fmt.Printf("Issue reformatting concert %v\n", err)
		}

		// Generate PDF tickets (injecting QR code)
		attachment := emailHandler.GenerateTicketPDF(order, *concert, true, redeemTicketURL)
		if err != nil {
			fmt.Printf("Unable to generate QR code: %s\n", err)
			return
		}

		// Email user with PDF attached
		err = emailHandler.SendEmail(order, attachment)
		if err != nil {
			fmt.Printf("Unable to send email: %s\n", err)
			return
		}
	}
	response.Body = "Success"
	response.StatusCode = 200
	return
}

// ===============================================================================================================================
// PUBLIC FUNCTIONS
// ===============================================================================================================================

func Handler(request events.APIGatewayProxyRequest) (response events.APIGatewayProxyResponse, err error) {
	fmt.Println("Handler called")
	sess, err := session.NewSession()
	if err != nil {
		fmt.Println(err)
		return
	}
	ddbsvc := dynamodb.New(sess)
	sessvc := sesv2.New(sess)
	concertsTable := os.Getenv("CONCERTS_TABLE")
	ordersTable := os.Getenv("ORDERS_TABLE")
	stripeSecret := os.Getenv("STRIPE_SECRET")
	senderAddress := os.Getenv("SENDER_ADDRESS")
	if concertsTable == "" || ordersTable == "" || senderAddress == "" || stripeSecret == "" {
		fmt.Println("CONCERTS_TABLE ORDERS_TABLE SENDER_ADDRESS and STRIPE_SECRET all need to be set as environment variables")
		return
	}
	fmt.Println("env vars all ok")
	dynamoHandler := ddbHandler.New(ddbsvc, concertsTable, ordersTable)
	stripeHandler := stripePaymentHandler.New(stripeSecret)
	sesHandler := sesEmailHandler.New(sessvc, senderAddress)

	return postPaymentHandler(request, dynamoHandler, stripeHandler, sesHandler)
}

func main() {
	lambda.Start(Handler)
}

// ===============================================================================================================================
// END PUBLIC FUNCTIONS
// ===============================================================================================================================
