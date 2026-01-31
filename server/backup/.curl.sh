#!/bin/bash
SERVER="https://77.92.144.37:17000"
TOKEN="93ef15c3aca08e1d65ba877b7811e3a597a0b3c5e91fdf5ec4b09381d6d08f54"
OUTDIR="./"

mkdir -p "$OUTDIR"

download_dir() {
    local path="$1"

    #echo "🔎︎ $path"
    echo "OO $path"

    # LIST isteği
    list=$(curl -k -s -X POST -H "Authorization: Bearer $TOKEN" "$SERVER/?action=list&path=$path" -H "Host: 127.0.0.1" )



    # Satır satır işlem
    while IFS= read -r line; do
        [ -z "$line" ] && continue

        # MD5 ve dosya adını ayır
        local md5=${line%% *}
        local name=${line#* }
        full="$path/$name"
        local local_file="$OUTDIR/$full"
        local local_md5="0"
        if [[ -f $local_file ]]; then
            # md5sum –b returns “<md5> <filename>”; cut out the first field.
            local_md5=$(md5sum -b "$local_file" | awk '{print $1}')
        fi
        
        
        if [[ -e "$full" && "$name" == ~* ]]; then
            echo "~~  $full"
            continue
        fi

        # klasör mü? slash ile bitiyor
        if [[ "$name" == */ ]]; then
            clean="${name%\/}"     # sondaki slash'i kaldır
            #echo "🗀  $path/$clean"
            echo "|-  $path/$clean"
            mkdir -p "$OUTDIR/$path/$clean"
            download_dir "$path/$clean"
        else
            if [[ "$md5" == "$local_md5" ]]; then
                #echo "⇋  $full"
                echo "==  $full"
            else
                #echo "🗎  $full"
                echo ">>  $full"
                curl -k -s -H "Authorization: Bearer $TOKEN" \
                    "$SERVER/?action=download&path=$full" \
                    -H "Host: 127.0.0.1" \
                    -o "$OUTDIR/$full"
            fi
        fi
    done <<< "$list"
}

download_dir "$1"