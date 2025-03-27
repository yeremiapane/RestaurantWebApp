# Midtrans API Integration

## Overview
This document describes the integration of Midtrans payment gateway in our application. The integration supports QRIS payment method and includes features like payment status monitoring, webhook handling, and transaction retry mechanism.

## Configuration

### Environment Variables
The following environment variables are required for Midtrans integration:

```env
MIDTRANS_SERVER_KEY=your_server_key
MIDTRANS_CLIENT_KEY=your_client_key
MIDTRANS_ENV=sandbox|production
MIDTRANS_MERCHANT_ID=your_merchant_id
MIDTRANS_MERCHANT_NAME=your_merchant_name
MIDTRANS_MERCHANT_EMAIL=your_merchant_email
MIDTRANS_MERCHANT_PHONE=your_merchant_phone
MIDTRANS_WEBHOOK_URL=your_webhook_url
```

## API Endpoints

### Create Payment
Creates a new payment transaction.

```http
POST /api/payments
Content-Type: application/json

{
    "amount": 10000,
    "order_id": 123,
    "payment_type": "qris"
}
```

Response:
```json
{
    "success": true,
    "data": {
        "id": 1,
        "order_id": 123,
        "amount": 10000,
        "status": "pending",
        "payment_type": "qris",
        "qr_code": "https://api.sandbox.midtrans.com/v2/qris/...",
        "expired_at": "2024-03-20T15:30:00Z"
    }
}
```

### Payment Callback
Webhook endpoint for Midtrans payment status updates.

```http
POST /api/payments/callback
Content-Type: application/json

{
    "order_id": "ORDER-123",
    "transaction_status": "settlement",
    "status_code": "200",
    "gross_amount": "10000",
    "signature_key": "valid_signature"
}
```

## Payment Status Flow

1. **Pending**
   - Initial state when payment is created
   - QR code is generated and displayed to customer
   - Payment is added to retry queue for status monitoring

2. **Success**
   - Payment is confirmed by Midtrans
   - Order status is updated to "paid"
   - Notification is sent to staff

3. **Failed**
   - Payment is rejected or fails
   - Order remains in original status
   - Notification is sent to staff

4. **Expired**
   - Payment QR code expires
   - Order remains in original status
   - Notification is sent to staff

## Security

### Signature Validation
All webhook callbacks from Midtrans are validated using SHA512 signature:

```go
signatureString := orderID + statusCode + grossAmount + serverKey
signature := sha512.Sum512([]byte(signatureString))
```

### Rate Limiting
Payment endpoints are rate limited to 10 requests per second to prevent abuse.

### Security Headers
The following security headers are added to payment endpoints:
- X-Frame-Options: DENY
- X-Content-Type-Options: nosniff
- X-XSS-Protection: 1; mode=block
- Content-Security-Policy: default-src 'self'
- Referrer-Policy: strict-origin-when-cross-origin
- Permissions-Policy: geolocation=(), microphone=(), camera=()

## Monitoring

### Payment Metrics
The system tracks the following metrics:
- Total transactions
- Successful payments
- Failed payments
- Pending payments
- Average response time

### Retry Mechanism
- Failed payments are automatically retried up to 5 times
- Retry interval is 5 minutes
- Each retry attempt is logged
- Final status is updated in database

## Error Handling

### Common Errors
1. **Invalid Amount**
   - Amount must be greater than 0
   - Amount must have at most 2 decimal places

2. **Invalid Order ID**
   - Order ID must exist in database
   - Order must not be already paid

3. **Invalid Payment Type**
   - Only "cash" and "qris" are supported

4. **API Errors**
   - Network errors
   - Invalid responses
   - Rate limiting

### Error Response Format
```json
{
    "error": "error_code",
    "message": "Human readable error message"
}
```

## Testing

### Unit Tests
Run unit tests for Midtrans service:
```bash
go test ./services -v
```

### Integration Tests
Run integration tests:
```bash
go test ./tests/integration -v
```

## Troubleshooting

### Common Issues
1. **Webhook Not Received**
   - Check webhook URL configuration
   - Verify server is accessible
   - Check logs for errors

2. **Invalid Signature**
   - Verify server key is correct
   - Check signature calculation
   - Validate request body format

3. **Payment Status Not Updated**
   - Check retry queue logs
   - Verify database connection
   - Check Midtrans API response

### Logging
Payment-related logs are stored in:
- Application logs: `logs/app.log`
- Payment monitor logs: `logs/payment_monitor.log`
- Midtrans API logs: `logs/midtrans.log`

## Best Practices

1. **Configuration**
   - Use environment variables for sensitive data
   - Validate configuration on startup
   - Use different keys for sandbox and production

2. **Error Handling**
   - Log all errors with context
   - Implement retry mechanism
   - Send notifications for critical errors

3. **Security**
   - Validate all inputs
   - Use HTTPS for webhooks
   - Implement rate limiting
   - Add security headers

4. **Monitoring**
   - Track payment metrics
   - Monitor retry queue
   - Set up alerts for failures

5. **Testing**
   - Write unit tests
   - Implement integration tests
   - Test error scenarios
   - Mock external API calls 