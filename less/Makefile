ifeq ($(shell which lessc),/usr/bin/lessc)
	LESSC=/usr/bin/lessc
else
	LESSC=node_modules/.bin/lessc
endif
export LESSC

CSSDIR=../static/css/

all :
	$(LESSC) app.less --clean-css="--s1 --advanced" $(CSSDIR)main.css

install : 
	./install-less.sh
	$(MAKE) all

clean :
	rm -f $(CSSDIR)main.css
