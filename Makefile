VERSION="0.2.23"
NAME="cascade"
KEEP=2

all: cascade deb

clean:
	rm -rf cascade
	[ -f ./pkg ] && ls -t ./pkg/*.deb | sed -e '1,$(KEEP)d' | xargs --no-run-if-empty -d "\n" rm -rf || true
	rm -rf ./build

cascade: clean
	go build

deb: cascade
	chmod 700 cascade
	mkdir -p ./build/usr/bin
	mkdir -p ./pkg
	cp cascade ./build/usr/bin
	fpm -t deb -s dir -n $(NAME) -v $(VERSION) -a amd64 --deb-user root --deb-group root -p ./pkg -C ./build .
