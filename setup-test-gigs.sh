#!/bin/bash

API_URL="http://localhost:8080/api/v1"

# Function to login and get token
login() {
    local username=$1
    local password=$2
    local response=$(curl -s -X POST "$API_URL/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$username\",\"password\":\"$password\"}")
    echo $response | grep -o '"token":"[^"]*' | grep -o '[^"]*$'
}

# Function to create a task
create_task() {
    local token=$1
    local title=$2
    local description=$3
    local deadline=$4
    local fee=$5
    
    curl -s -X POST "$API_URL/tasks" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "{
            \"title\":\"$title\",
            \"description\":\"$description\",
            \"deadline\":\"$deadline\",
            \"fee\":$fee
        }"
}

echo "Logging in users and creating gigs..."

# Login Alice and create her gigs
ALICE_TOKEN=$(login "alice_dev" "alice123")
echo "Alice logged in"

# Calculate deadlines
WEEK_FROM_NOW=$(date -d "+7 days" --iso-8601=seconds)
THREE_DAYS=$(date -d "+3 days" --iso-8601=seconds)
FIVE_DAYS=$(date -d "+5 days" --iso-8601=seconds)
TEN_DAYS=$(date -d "+10 days" --iso-8601=seconds)
TWO_WEEKS=$(date -d "+14 days" --iso-8601=seconds)

# Alice's gigs
create_task "$ALICE_TOKEN" \
    "Build React Dashboard Component" \
    "Need a responsive dashboard component built with React and Tailwind CSS. Should include charts, stats cards, and recent activity feed." \
    "$WEEK_FROM_NOW" \
    500

create_task "$ALICE_TOKEN" \
    "Fix Authentication Bug in Node.js API" \
    "JWT tokens are not refreshing properly. Need someone experienced with Node.js and JWT authentication to debug and fix the issue." \
    "$THREE_DAYS" \
    200

echo "Alice's gigs created"

# Login Bob and create his gigs
BOB_TOKEN=$(login "bob_designer" "bob123")
echo "Bob logged in"

create_task "$BOB_TOKEN" \
    "Design Logo for Tech Startup" \
    "Looking for a modern, minimalist logo design for a tech startup. Should work well in both light and dark modes." \
    "$FIVE_DAYS" \
    300

create_task "$BOB_TOKEN" \
    "Create UI/UX for Mobile App" \
    "Need complete UI/UX design for a fitness tracking mobile app. Includes wireframes, mockups, and design system." \
    "$TWO_WEEKS" \
    1200

echo "Bob's gigs created"

# Login Charlie and create his gigs
CHARLIE_TOKEN=$(login "charlie_writer" "charlie123")
echo "Charlie logged in"

create_task "$CHARLIE_TOKEN" \
    "Write Technical Blog Posts" \
    "Need 5 technical blog posts about cloud computing and DevOps practices. Each post should be 1000-1500 words." \
    "$TEN_DAYS" \
    400

create_task "$CHARLIE_TOKEN" \
    "Edit and Proofread API Documentation" \
    "Review and improve API documentation for clarity and completeness. Experience with technical writing required." \
    "$FIVE_DAYS" \
    150

echo "Charlie's gigs created"

# Login Diana and create her gigs
DIANA_TOKEN=$(login "diana_coder" "diana123")
echo "Diana logged in"

create_task "$DIANA_TOKEN" \
    "Implement Payment Integration" \
    "Integrate Stripe payment processing into existing e-commerce platform. Must handle subscriptions and one-time payments." \
    "$WEEK_FROM_NOW" \
    800

create_task "$DIANA_TOKEN" \
    "Optimize Database Queries" \
    "PostgreSQL database needs performance optimization. Several queries taking too long. Need someone with strong SQL skills." \
    "$THREE_DAYS" \
    350

echo "Diana's gigs created"

echo ""
echo "=== TEST DATA CREATED ==="
echo ""
echo "4 Users created with the following credentials:"
echo ""
echo "1. alice_dev / alice123 - Full Stack Developer"
echo "2. bob_designer / bob123 - UI/UX Designer"
echo "3. charlie_writer / charlie123 - Technical Writer"
echo "4. diana_coder / diana123 - Backend Developer"
echo ""
echo "Each user has created 2 gigs with different deadlines and fees."
echo ""
echo "You can now login with any of these accounts to:"
echo "- View the dashboard with created gigs"
echo "- Browse and apply for other users' gigs"
echo "- Send messages about gigs"
echo "- Accept/decline applications"
echo ""