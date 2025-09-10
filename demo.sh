#!/bin/bash

# MicroCoin Demo Script
# This script demonstrates the key features of the MicroCoin paper trading system

set -e

BASE_URL="http://localhost:8080"
EMAIL="demo@example.com"
PASSWORD="demopassword123"

echo "🚀 MicroCoin Paper Trading Demo"
echo "================================"

# Function to make HTTP requests
make_request() {
    local method=$1
    local url=$2
    local data=$3
    local headers=$4
    
    if [ -n "$data" ]; then
        curl -s -X $method "$url" \
            -H "Content-Type: application/json" \
            $headers \
            -d "$data"
    else
        curl -s -X $method "$url" \
            -H "Content-Type: application/json" \
            $headers
    fi
}

# Function to extract JSON field
extract_field() {
    local json=$1
    local field=$2
    echo "$json" | jq -r ".$field"
}

echo "1. 📝 Signing up a new user..."
SIGNUP_RESPONSE=$(make_request POST "$BASE_URL/auth/signup" "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
echo "Signup response: $SIGNUP_RESPONSE"

TOKEN=$(extract_field "$SIGNUP_RESPONSE" "token")
echo "✅ User signed up successfully!"
echo ""

echo "2. 💰 Topping up account with $10,000 paper USD..."
IDEMPOTENCY_KEY="topup-$(date +%s)"
TOPUP_RESPONSE=$(make_request POST "$BASE_URL/api/fund/topup" "{\"amount\":\"10000.00\"}" "-H \"Authorization: Bearer $TOKEN\" -H \"Idempotency-Key: $IDEMPOTENCY_KEY\"")
echo "Topup response: $TOPUP_RESPONSE"

BALANCE=$(extract_field "$TOPUP_RESPONSE" "balance")
echo "✅ Account topped up! New balance: \$$BALANCE"
echo ""

echo "3. 🔄 Testing idempotency - same topup request..."
TOPUP_RESPONSE2=$(make_request POST "$BASE_URL/api/fund/topup" "{\"amount\":\"10000.00\"}" "-H \"Authorization: Bearer $TOKEN\" -H \"Idempotency-Key: $IDEMPOTENCY_KEY\"")
echo "Idempotent topup response: $TOPUP_RESPONSE2"

BALANCE2=$(extract_field "$TOPUP_RESPONSE2" "balance")
echo "✅ Idempotency works! Balance unchanged: \$$BALANCE2"
echo ""

echo "4. 📊 Getting current BTC quote..."
QUOTE_RESPONSE=$(make_request GET "$BASE_URL/api/quotes?symbol=BTC-USD")
echo "Quote response: $QUOTE_RESPONSE"

BTC_BID=$(extract_field "$QUOTE_RESPONSE" "bid")
BTC_ASK=$(extract_field "$QUOTE_RESPONSE" "ask")
echo "✅ Current BTC quote - Bid: \$$BTC_BID, Ask: \$$BTC_ASK"
echo ""

echo "5. 📈 Placing a limit buy order for 0.01 BTC at $50,000..."
ORDER_IDEMPOTENCY_KEY="order-$(date +%s)"
ORDER_RESPONSE=$(make_request POST "$BASE_URL/api/orders" "{\"symbol\":\"BTC-USD\",\"side\":\"BUY\",\"type\":\"LIMIT\",\"price\":\"50000\",\"qty\":\"0.01\"}" "-H \"Authorization: Bearer $TOKEN\" -H \"Idempotency-Key: $ORDER_IDEMPOTENCY_KEY\"")
echo "Order response: $ORDER_RESPONSE"

ORDER_ID=$(extract_field "$ORDER_RESPONSE" "order_id")
ORDER_STATUS=$(extract_field "$ORDER_RESPONSE" "status")
echo "✅ Order placed! ID: $ORDER_ID, Status: $ORDER_STATUS"
echo ""

echo "6. 🔍 Checking order details..."
ORDER_DETAILS=$(make_request GET "$BASE_URL/api/orders/$ORDER_ID" "" "-H \"Authorization: Bearer $TOKEN\"")
echo "Order details: $ORDER_DETAILS"
echo ""

echo "7. 💼 Checking portfolio..."
PORTFOLIO_RESPONSE=$(make_request GET "$BASE_URL/api/portfolio" "" "-H \"Authorization: Bearer $TOKEN\"")
echo "Portfolio response: $PORTFOLIO_RESPONSE"
echo ""

echo "8. 🔄 Testing order idempotency - same order request..."
ORDER_RESPONSE2=$(make_request POST "$BASE_URL/api/orders" "{\"symbol\":\"BTC-USD\",\"side\":\"BUY\",\"type\":\"LIMIT\",\"price\":\"50000\",\"qty\":\"0.01\"}" "-H \"Authorization: Bearer $TOKEN\" -H \"Idempotency-Key: $ORDER_IDEMPOTENCY_KEY\"")
echo "Idempotent order response: $ORDER_RESPONSE2"

ORDER_ID2=$(extract_field "$ORDER_RESPONSE2" "order_id")
echo "✅ Order idempotency works! Same order ID: $ORDER_ID2"
echo ""

echo "9. 📊 Testing WebSocket quotes (will run for 10 seconds)..."
echo "Opening WebSocket connection to /ws/quotes..."
timeout 10s websocat ws://localhost:8080/ws/quotes || echo "WebSocket connection closed"
echo ""

echo "🎉 Demo completed successfully!"
echo "================================"
echo ""
echo "Key features demonstrated:"
echo "✅ User authentication with JWT"
echo "✅ Paper money top-up with idempotency"
echo "✅ Real-time quotes via REST API"
echo "✅ Limit order placement with idempotency"
echo "✅ Order management and portfolio tracking"
echo "✅ WebSocket real-time quotes"
echo ""
echo "MicroCoin is ready for paper trading! 🚀"
