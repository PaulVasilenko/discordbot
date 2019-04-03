build:
	GOOS=linux GOARCH=386 go build -o bot/discordbot ../discordbot/bot.go
	GOOS=linux GOARCH=386 go build -o bot/chrisify ../../chrisify/...
	docker build -t discordbot bot/
	docker build -t discordbot_mysql mysql/

run:
	docker-compose start