#!/bin/bash

API_URL="http://localhost:8080/api/v1"

echo "Creating test users..."

# Create Alice
curl -X POST "$API_URL/register" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice_dev",
    "email": "alice@example.com",
    "password": "alice123",
    "full_name": "Alice Johnson"
  }'
echo -e "\n"

# Create Bob
curl -X POST "$API_URL/register" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "bob_designer",
    "email": "bob@example.com",
    "password": "bob123",
    "full_name": "Bob Smith"
  }'
echo -e "\n"

# Create Charlie
curl -X POST "$API_URL/register" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "charlie_writer",
    "email": "charlie@example.com",
    "password": "charlie123",
    "full_name": "Charlie Brown"
  }'
echo -e "\n"

# Create Diana
curl -X POST "$API_URL/register" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "diana_coder",
    "email": "diana@example.com",
    "password": "diana123",
    "full_name": "Diana Prince"
  }'
echo -e "\n"

echo "Test users created!"
echo ""
echo "=== LOGIN CREDENTIALS ==="
echo "Username: alice_dev"
echo "Password: alice123"
echo "---"
echo "Username: bob_designer"
echo "Password: bob123"
echo "---"
echo "Username: charlie_writer"
echo "Password: charlie123"
echo "---"
echo "Username: diana_coder"
echo "Password: diana123"
echo ""