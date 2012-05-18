#!/bin/bash

RESFOLDER=web
RESFILE=webresources.go

echo $RESFILE

echo extracting web resources and storing them in a go map
rm $RESFILE
echo package main >> $RESFILE
echo "// GENERATED FILE: Append here all the Make generated resources" >> $RESFILE
echo "// webresources[\"index.html\"] = \`etc..\`" >> $RESFILE
echo "// webresources[\"css/style.css\"] = \`etc..\` " >> $RESFILE
echo "func setVariables() {"  >> $RESFILE

LIST=$(find $RESFOLDER -type f | egrep '\.(html|css|js)$')
for i in $LIST; do
	RESNAME=$(echo $i | sed -e "s/${RESFOLDER}\///g")
	echo writing: $RESNAME
	echo webresources[\"$RESNAME\"] = \` >> $RESFILE
	cat "$i" | sed -e 's/`/\\u0060/g' >> $RESFILE
	echo \` >> $RESFILE
     # NEWNAME=$(ls "$i" | sed -e 's/html/php/')
     # cat beginfile > "$NEWNAME"
     # cat "$i" | sed -e '1,25d' | tac | sed -e '1,21d'| tac >> "$NEWNAME"
     # cat endfile >> "$NEWNAME"
done

echo "return }"  >> $RESFILE
