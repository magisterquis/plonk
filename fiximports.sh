#!/bin/ksh

for f in $@; do
        #sed -Ei.bak '/	"github.com\/magisterquis\/plonk_rewrite/d' "$f"
        sed -Ei.bak 's/"github.com\/magisterquis\/plonk_rewrite/"github.com\/magisterquis\/plonk/' "$f"
        goimports -d "$f"
        echo "$f"
        diff "$f" "$f.bak"
done
