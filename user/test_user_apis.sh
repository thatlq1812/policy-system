#!/bin/bash

# User Service API Testing Script
# Tests all endpoints with various error cases

HOST="localhost:50052"
SERVICE="user.UserService"

echo "======================================"
echo "User Service API Testing"
echo "======================================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to test endpoint
test_endpoint() {
    local test_name=$1
    local method=$2
    local data=$3
    local expected_error=$4
    
    echo "Testing: $test_name"
    echo "Method: $method"
    echo "Data: $data"
    
    result=$(grpcurl -plaintext -d "$data" $HOST $SERVICE/$method 2>&1)
    exit_code=$?
    
    if [ $exit_code -eq 0 ]; then
        if [ -z "$expected_error" ]; then
            echo -e "${GREEN}✓ SUCCESS${NC}"
            echo "$result" | head -20
        else
            echo -e "${RED}✗ FAILED - Expected error but got success${NC}"
            echo "$result"
        fi
    else
        if [ -n "$expected_error" ]; then
            if echo "$result" | grep -q "$expected_error"; then
                echo -e "${GREEN}✓ ERROR AS EXPECTED: $expected_error${NC}"
            else
                echo -e "${RED}✗ FAILED - Wrong error${NC}"
                echo "Expected: $expected_error"
                echo "Got: $result"
            fi
        else
            echo -e "${RED}✗ FAILED - Unexpected error${NC}"
            echo "$result"
        fi
    fi
    echo ""
    echo "--------------------------------------"
    echo ""
}

echo "======================================"
echo "1. REGISTER API TESTS"
echo "======================================"
echo ""

# Test 1.1: Valid registration
test_endpoint \
    "1.1 Valid Registration" \
    "Register" \
    '{"phone_number":"0901234567","password":"password123","name":"Test User","platform_role":"Client"}' \
    ""

# Test 1.2: Missing phone number
test_endpoint \
    "1.2 Missing Phone Number" \
    "Register" \
    '{"password":"password123","name":"Test User","platform_role":"Client"}' \
    "InvalidArgument"

# Test 1.3: Short password
test_endpoint \
    "1.3 Short Password" \
    "Register" \
    '{"phone_number":"0901234568","password":"12345","name":"Test User","platform_role":"Client"}' \
    "InvalidArgument"

# Test 1.4: Invalid platform role
test_endpoint \
    "1.4 Invalid Platform Role" \
    "Register" \
    '{"phone_number":"0901234569","password":"password123","name":"Test User","platform_role":"InvalidRole"}' \
    "InvalidArgument"

# Test 1.5: Duplicate phone number
test_endpoint \
    "1.5 Duplicate Phone Number" \
    "Register" \
    '{"phone_number":"0901234567","password":"password123","name":"Test User 2","platform_role":"Client"}' \
    "AlreadyExists"

echo ""
echo "======================================"
echo "2. LOGIN API TESTS"
echo "======================================"
echo ""

# Test 2.1: Valid login
test_endpoint \
    "2.1 Valid Login" \
    "Login" \
    '{"phone_number":"0901234567","password":"password123"}' \
    ""

# Test 2.2: Missing phone number
test_endpoint \
    "2.2 Missing Phone Number" \
    "Login" \
    '{"password":"password123"}' \
    "InvalidArgument"

# Test 2.3: Wrong password
test_endpoint \
    "2.3 Wrong Password" \
    "Login" \
    '{"phone_number":"0901234567","password":"wrongpassword"}' \
    "Unauthenticated"

# Test 2.4: Non-existent user
test_endpoint \
    "2.4 Non-existent User" \
    "Login" \
    '{"phone_number":"0909999999","password":"password123"}' \
    "Unauthenticated"

echo ""
echo "======================================"
echo "3. REFRESH TOKEN API TESTS"
echo "======================================"
echo ""

# First, get a valid refresh token
echo "Getting valid refresh token..."
login_result=$(grpcurl -plaintext -d '{"phone_number":"0901234567","password":"password123"}' $HOST $SERVICE/Login 2>&1)
refresh_token=$(echo "$login_result" | grep -o '"refreshToken": "[^"]*"' | cut -d'"' -f4)

if [ -n "$refresh_token" ]; then
    echo "Got refresh token: ${refresh_token:0:20}..."
    echo ""
    
    # Test 3.1: Valid refresh
    test_endpoint \
        "3.1 Valid Refresh Token" \
        "RefreshToken" \
        "{\"refresh_token\":\"$refresh_token\"}" \
        ""
    
    # Test 3.2: Invalid refresh token
    test_endpoint \
        "3.2 Invalid Refresh Token" \
        "RefreshToken" \
        '{"refresh_token":"invalid-token-12345"}' \
        "Unauthenticated"
    
    # Test 3.3: Empty refresh token
    test_endpoint \
        "3.3 Empty Refresh Token" \
        "RefreshToken" \
        '{"refresh_token":""}' \
        "InvalidArgument"
else
    echo -e "${RED}Failed to get refresh token for testing${NC}"
fi

echo ""
echo "======================================"
echo "4. LOGOUT API TESTS"
echo "======================================"
echo ""

# Get tokens for logout test
echo "Getting tokens for logout test..."
login_result=$(grpcurl -plaintext -d '{"phone_number":"0901234567","password":"password123"}' $HOST $SERVICE/Login 2>&1)
refresh_token=$(echo "$login_result" | grep -o '"refreshToken": "[^"]*"' | cut -d'"' -f4)
access_token=$(echo "$login_result" | grep -o '"accessToken": "[^"]*"' | cut -d'"' -f4)

if [ -n "$refresh_token" ]; then
    echo "Got tokens for logout"
    echo ""
    
    # Test 4.1: Logout with both tokens
    test_endpoint \
        "4.1 Logout With Both Tokens" \
        "Logout" \
        "{\"refresh_token\":\"$refresh_token\",\"access_token\":\"$access_token\"}" \
        ""
    
    # Test 4.2: Logout with already revoked token
    test_endpoint \
        "4.2 Logout With Revoked Token" \
        "Logout" \
        "{\"refresh_token\":\"$refresh_token\"}" \
        ""
    
    # Test 4.3: Empty refresh token
    test_endpoint \
        "4.3 Empty Refresh Token" \
        "Logout" \
        '{"refresh_token":""}' \
        "InvalidArgument"
fi

echo ""
echo "======================================"
echo "5. USER PROFILE TESTS"
echo "======================================"
echo ""

# Get user ID from login
login_result=$(grpcurl -plaintext -d '{"phone_number":"0901234567","password":"password123"}' $HOST $SERVICE/Login 2>&1)
user_id=$(echo "$login_result" | grep -o '"id": "[^"]*"' | cut -d'"' -f4)

if [ -n "$user_id" ]; then
    echo "Got user ID: $user_id"
    echo ""
    
    # Test 5.1: Get valid profile
    test_endpoint \
        "5.1 Get Valid Profile" \
        "GetUserProfile" \
        "{\"user_id\":\"$user_id\"}" \
        ""
    
    # Test 5.2: Get non-existent profile
    test_endpoint \
        "5.2 Get Non-existent Profile" \
        "GetUserProfile" \
        '{"user_id":"00000000-0000-0000-0000-000000000000"}' \
        "NotFound"
    
    # Test 5.3: Empty user ID
    test_endpoint \
        "5.3 Empty User ID" \
        "GetUserProfile" \
        '{"user_id":""}' \
        "InvalidArgument"
    
    # Test 5.4: Update profile
    test_endpoint \
        "5.4 Update Profile" \
        "UpdateUserProfile" \
        "{\"user_id\":\"$user_id\",\"name\":\"Updated Name\"}" \
        ""
    
    # Test 5.5: Update with invalid user ID
    test_endpoint \
        "5.5 Update Invalid User ID" \
        "UpdateUserProfile" \
        '{"user_id":"invalid-id","name":"Test"}' \
        "NotFound"
fi

echo ""
echo "======================================"
echo "6. CHANGE PASSWORD TESTS"
echo "======================================"
echo ""

if [ -n "$user_id" ]; then
    # Test 6.1: Valid password change
    test_endpoint \
        "6.1 Valid Password Change" \
        "ChangePassword" \
        "{\"user_id\":\"$user_id\",\"old_password\":\"password123\",\"new_password\":\"newpassword123\"}" \
        ""
    
    # Change back to original
    grpcurl -plaintext -d "{\"user_id\":\"$user_id\",\"old_password\":\"newpassword123\",\"new_password\":\"password123\"}" $HOST $SERVICE/ChangePassword > /dev/null 2>&1
    
    # Test 6.2: Wrong old password
    test_endpoint \
        "6.2 Wrong Old Password" \
        "ChangePassword" \
        "{\"user_id\":\"$user_id\",\"old_password\":\"wrongpassword\",\"new_password\":\"newpassword123\"}" \
        "Unauthenticated"
    
    # Test 6.3: Short new password
    test_endpoint \
        "6.3 Short New Password" \
        "ChangePassword" \
        "{\"user_id\":\"$user_id\",\"old_password\":\"password123\",\"new_password\":\"123\"}" \
        "InvalidArgument"
    
    # Test 6.4: Empty fields
    test_endpoint \
        "6.4 Empty Fields" \
        "ChangePassword" \
        '{"user_id":"","old_password":"","new_password":""}' \
        "InvalidArgument"
fi

echo ""
echo "======================================"
echo "7. ADMIN - LIST USERS TESTS"
echo "======================================"
echo ""

# Test 7.1: List users - valid
test_endpoint \
    "7.1 List Users - Valid" \
    "ListUsers" \
    '{"page":1,"page_size":10}' \
    ""

# Test 7.2: List users with filter
test_endpoint \
    "7.2 List Users With Filter" \
    "ListUsers" \
    '{"page":1,"page_size":10,"platform_role":"Client"}' \
    ""

# Test 7.3: Invalid page number
test_endpoint \
    "7.3 Invalid Page Number" \
    "ListUsers" \
    '{"page":0,"page_size":10}' \
    ""

echo ""
echo "======================================"
echo "8. ADMIN - SEARCH USERS TESTS"
echo "======================================"
echo ""

# Test 8.1: Search users
test_endpoint \
    "8.1 Search Users By Name" \
    "SearchUsers" \
    '{"query":"Test","limit":10}' \
    ""

# Test 8.2: Empty query
test_endpoint \
    "8.2 Empty Query" \
    "SearchUsers" \
    '{"query":"","limit":10}' \
    "InvalidArgument"

echo ""
echo "======================================"
echo "9. GET USER STATS TESTS"
echo "======================================"
echo ""

# Test 9.1: Get stats
test_endpoint \
    "9.1 Get User Stats" \
    "GetUserStats" \
    '{}' \
    ""

echo ""
echo "======================================"
echo "10. TOKEN BLACKLIST TESTS"
echo "======================================"
echo ""

# Test 10.1: Check non-existent JTI
test_endpoint \
    "10.1 Check Non-existent JTI" \
    "IsTokenBlacklisted" \
    '{"jti":"test-jti-12345"}' \
    ""

# Test 10.2: Empty JTI
test_endpoint \
    "10.2 Empty JTI" \
    "IsTokenBlacklisted" \
    '{"jti":""}' \
    "InvalidArgument"

echo ""
echo "======================================"
echo "TESTING COMPLETE"
echo "======================================"
