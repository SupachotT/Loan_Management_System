# Start Database Applicant
openDB:
	docker exec -ti LMS_Container createdb -U Admin LMS_LoanApplicantsDB
	docker exec -ti LMS_Container psql -U Admin

clean:
	docker stop LMS_Container
	docker rm LMS_Container