#!/bin/bash
# Script to manually retry booking notifications for existing bookings
# that were created before environment variables were configured

set -e

CHAT_API_URL="http://localhost:8082/api/v1"
INTERNAL_API_KEY="your-internal-api-key-change-in-production"

echo "🔄 Retrying booking notifications..."
echo ""

# Get all booking requests that haven't been notified
BOOKINGS=$(docker exec myguy-postgres-db-1 psql -U postgres -d my_guy_store -t -c "
  SELECT
    br.id,
    br.item_id,
    br.requester_id,
    si.seller_id,
    si.title,
    COALESCE((SELECT url FROM item_images WHERE item_id = si.id ORDER BY \"order\" LIMIT 1), '') as image
  FROM booking_requests br
  JOIN store_items si ON br.item_id = si.id
  WHERE br.chat_notified = false
  AND br.status = 'pending'
  ORDER BY br.created_at;
")

if [ -z "$BOOKINGS" ]; then
  echo "✅ No booking requests need notification retry"
  exit 0
fi

echo "Found booking requests to notify:"
echo "$BOOKINGS"
echo ""

# Process each booking
echo "$BOOKINGS" | while IFS='|' read -r booking_id item_id requester_id seller_id title image; do
  # Trim whitespace
  booking_id=$(echo $booking_id | xargs)
  item_id=$(echo $item_id | xargs)
  requester_id=$(echo $requester_id | xargs)
  seller_id=$(echo $seller_id | xargs)
  title=$(echo $title | xargs)
  image=$(echo $image | xargs)

  echo "📤 Sending notification for booking #$booking_id (item: $title)"

  # Send notification to chat service
  HTTP_CODE=$(curl -s -w "%{http_code}" -o /tmp/booking_response.txt -X POST "${CHAT_API_URL}/internal/booking-created" \
    -H "Content-Type: application/json" \
    -H "X-Internal-API-Key: ${INTERNAL_API_KEY}" \
    -d "{
      \"bookingId\": $booking_id,
      \"itemId\": $item_id,
      \"itemTitle\": \"$title\",
      \"itemImage\": \"$image\",
      \"buyerId\": $requester_id,
      \"sellerId\": $seller_id
    }")

  BODY=$(cat /tmp/booking_response.txt 2>/dev/null || echo "")

  if [ "$HTTP_CODE" = "200" ]; then
    echo "  ✅ Notification sent successfully"
    # Update chat_notified flag
    docker exec myguy-postgres-db-1 psql -U postgres -d my_guy_store -c \
      "UPDATE booking_requests SET chat_notified = true WHERE id = $booking_id;" > /dev/null
    echo "  ✅ Database updated"
  else
    echo "  ❌ Failed with HTTP $HTTP_CODE"
    echo "  Response: $BODY"
  fi

  echo ""
done

echo "✅ Notification retry completed!"
