# Start server postgres
service:
	docker pull supachott/postgres
	docker run --name LMS_Container -e POSTGRES_USER=Admin -e POSTGRES_PASSWORD=Password -p 5432:5432 -d supachott/postgres

clean:
	docker stop LMS_Container
	docker rm LMS_Container

rmImage:
	docker rmi postgres:latest