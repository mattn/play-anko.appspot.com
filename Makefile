.PHONY: deploy

deploy:
	rm -rf github.com
	git clone --depth 1 https://github.com/mattn/anko github.com/mattn/anko
	cp github.com/mattn/anko/.git/refs/heads/master VERSION
	gcloud -q app deploy --project play-anko 
