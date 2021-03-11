.PHONY: deploy

deploy:
	curl -s https://api.github.com/repos/mattn/anko/commits/master | jq -r .sha > VERSION
	gcloud -q app deploy --project play-anko 
