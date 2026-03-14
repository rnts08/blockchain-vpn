# Billing System Architecture

This document describes the billing system architecture for BlockchainVPN.

## Overview

The billing system supports three pricing models:

1. **Session-based** (default): Flat fee per session
2. **Time-based**: Price per time unit (e.g., per minute, per hour)
3. **Data-based**: Price per data unit (e.g., per MB, per GB)

## Core Components

### UsageMeter (`internal/tunnel/usage.go`)

Tracks time and data usage for a VPN session according to the provider's pricing model.

**Key Fields:**
- `pricingMethod`: 0=session, 1=time, 2=data
- `pricePerUnit`: Price per billing unit in satoshis
- `timeUnitSecs`: Time duration per billing cycle (for time-based)
- `dataUnitBytes`: Data amount per billing cycle (for data-based)

**Key Methods:**
- `CurrentCost()`: Calculates total cost based on usage
- `ShouldRenewPayment()`: Determines when to request additional payment
- `AddTraffic()`: Increments byte counters

### SpendingManager (`internal/tunnel/credit_manager.go`)

Manages client spending, enforces limits, and handles auto-recharge.

**Key Features:**
- Total spending limits (daily/periodic)
- Session spending limits
- Warning thresholds (e.g., alert at 80% of limit)
- Auto-disconnect when limit reached
- Auto-recharge from wallet

### Payment Verification (`internal/blockchain/payment.go`)

Verifies and processes blockchain payments.

**Functions:**
- `VerifyPaymentInput()`: Validates payment meets advertised price
- `SendPayment()`: Creates and broadcasts payment transaction

## Billing Flow

### Time-Based Billing

1. Client connects to provider
2. UsageMeter starts tracking time
3. Every billing period (e.g., 60 seconds), CurrentCost() increases
4. When ShouldRenewPayment() returns true, client sends new payment
5. Provider verifies payment before continuing service

### Data-Based Billing

1. Client connects to provider
2. UsageMeter tracks bytes sent/received
3. Every time data unit is consumed (e.g., 1MB), cost increases
4. When threshold reached, client sends new payment
5. Provider verifies payment before continuing

### Payment Verification

1. Client creates OP_RETURN transaction with public key
2. Provider scans blockchain for payments to their address
3. Provider decodes OP_RETURN to verify client identity
4. Service is granted upon valid payment

## Configuration

### Provider Configuration

Providers set pricing in their announcement:
- `Price`: Base price in satoshis
- `PricingMethod`: 0 (session), 1 (time), or 2 (data)
- `TimeUnitSecs`: For time-based billing
- `DataUnitBytes`: For data-based billing
- `SessionTimeoutSecs`: Maximum session duration

### Client Configuration

Clients can set spending controls:
- `SpendingLimitEnabled`: Enable/disable spending limits
- `SpendingLimitSats`: Maximum total spending
- `MaxSessionSpendingSats`: Per-session spending limit
- `AutoDisconnectOnLimit`: Disconnect when limit reached
- `AutoRechargeEnabled`: Auto-pay when balance low
- `AutoRechargeThreshold`: Threshold for auto-recharge
