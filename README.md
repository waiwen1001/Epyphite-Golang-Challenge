Step 1 [install go project dependencies] : go mod tidy
Step 2 [use docker install postgresql] : docker install postgresql using port 5432
# docker run --name some-postgres -e POSTGRES_PASSWORD=bikepassword -p 5432:5432 -d postgres
Step 3 [serve project] : go run .

API doc :
API route required static token : [name : token] [value : bike001]
For more info please refer bike.yaml

Unit test : 
Run command : go test ./...