# WORKINDIA
inshorts application
Implemented user sigup, login, POST request to post news in DB and GET request to read all the news in descending order of publish_date and upvotes

Made server on port 9000 using golang to handle all the GET POST PUT DELETE requests from curl

Sample curl requests-
Curl requests-

Signup- 
curl -X POST http://localhost:9000/api/signup \
    -H "Content-Type: application/json" \
    -d '{"username": "example_user", "password": "example_password", "email": "user@example.com"}'

#Handles the errors related to repeated email id's being registered displays " Email already exists"
Try to avoid this error- 
curl -X POST http://localhost:9000/api/signup \
    -H "Content-Type: application/json" \
    -d '{"username": "new_user", "password": "password", "email": "guest@example.com"}'



Login-
curl -X POST http://localhost:9000/api/login \
    -H "Content-Type: application/json" \
    -d '{"username": "example_user", "password": "example_password"}'







POST- feed
curl -X POST http://localhost:9000/api/shorts/create \
-H "Content-Type: application/json" \
-d '{
    "category": "news",
    "title": "New news!",
    "author": "writer",
    "publish_date": "2023-01-01T16:00:00Z",
    "content": "Lorem ipsum ...",
    "actual_content_link": "http://instagram.com/sorry",
    "image": "",
    "votes": {
        "upvote": 20,,
        "downvote": 10
    }
}'





GET-

curl -X GET http://localhost:9000/api/shorts/feed

