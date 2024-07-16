# Start server postgres
service:
	docker pull supachott/postgres
	docker run --name LMS_Container -e POSTGRES_USER=Admin -e POSTGRES_PASSWORD=Password -p 5432:5432 -d supachott/postgres

openDB:
	docker exec -ti LMS_Container createdb -U Admin LMS_LoanApplicantsDB
	docker exec -ti LMS_Container psql -U Admin

clean:
	docker stop LMS_Container
	docker rm LMS_Container
	docker rmi supachott/postgres

rmImage:
	docker rmi supachott/postgres

runGo:
	go run ./main.go