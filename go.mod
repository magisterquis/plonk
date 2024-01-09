module github.com/magisterquis/plonk

go 1.21.3

require (
	github.com/magisterquis/simpleshsplit v0.0.0-20230820155114-587a91c66e4e
	golang.org/x/exp v0.0.0-20231206192017-f3f8817b8deb
	golang.org/x/sys v0.15.0
	golang.org/x/term v0.15.0
)

require github.com/magisterquis/mqd v0.0.0-20231010173215-36e6cea04f08

require golang.org/x/sync v0.5.0

require (
	golang.org/x/net v0.19.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

replace github.com/magisterquis/eztls => ../eztls

replace github.com/magisterquis/mqd => ../mqd
