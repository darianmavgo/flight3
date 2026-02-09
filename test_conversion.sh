#!/bin/bash

# Test Flight3 /api/convert endpoint

echo "Testing Flight3 Conversion API"
echo "================================"
echo ""

# Flight3 server URL
FLIGHT3_URL="http://127.0.0.1:8095"
TEST_FILE="/tmp/test_data.csv"
OUTPUT_FILE="/tmp/converted_result.db"

# Check if test file exists
if [ ! -f "$TEST_FILE" ]; then
    echo "❌ Test file not found: $TEST_FILE"
    exit 1
fi

echo "📋 Test file: $TEST_FILE"
echo "🌐 Flight3 URL: $FLIGHT3_URL"
echo ""

# Test the /api/convert endpoint
echo "🚀 Sending conversion request..."
HTTP_CODE=$(curl -w "%{http_code}" -o "$OUTPUT_FILE" -F "file=@$TEST_FILE" "$FLIGHT3_URL/api/convert" 2>/dev/null)

echo "📊 HTTP Status Code: $HTTP_CODE"
echo ""

if [ "$HTTP_CODE" == "200" ]; then
    echo "✅ Conversion successful!"
    
    # Check if output file exists and is valid SQLite
    if [ -f "$OUTPUT_FILE" ]; then
        FILE_SIZE=$(stat -f%z "$OUTPUT_FILE" 2>/dev/null || stat -c%s "$OUTPUT_FILE" 2>/dev/null)
        echo "📦 Output file size: $FILE_SIZE bytes"
        
        # Try to query the SQLite database
        echo ""
        echo "🔍 Testing SQLite database..."
        echo ""
        sqlite3 "$OUTPUT_FILE" "SELECT * FROM tb0 LIMIT 5;"
        
        echo ""
        echo "✅ All tests passed!"
    else
        echo "❌ Output file not created"
        exit 1
    fi
else
    echo "❌ Conversion failed"
    echo ""
    echo "Response:"
    cat "$OUTPUT_FILE"
    echo ""
    exit 1
fi
