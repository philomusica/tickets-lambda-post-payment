module github.com/philomusica/tickets-lambda-post-payment

go 1.20

require (
	github.com/aws/aws-lambda-go v1.38.0
	github.com/aws/aws-sdk-go v1.44.213
	github.com/philomusica/tickets-lambda-basket-service v1.3.0
	github.com/philomusica/tickets-lambda-get-concerts v1.7.0
	github.com/signintech/gopdf v0.16.1
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	github.com/stripe/stripe-go/v74 v74.10.0
	gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df
)

require (
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/philomusica/tickets-lambda-process-payment v1.2.4 // indirect
	github.com/phpdave11/gofpdi v1.0.14-0.20211212211723-1f10f9844311 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
)
