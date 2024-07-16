# Start Database Apply Loan
openDB:
	docker exec -ti LMS_Container createdb -U Admin LMS_ApplyLoanDB
	docker exec -ti LMS_Container psql -U Admin

clean:
	docker stop LMS_Container
	docker rm LMS_Container