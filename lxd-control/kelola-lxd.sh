#!/bin/bash

usage() {
    echo "Penggunaan:"
    echo "  $0 provision <master_container> <profile> <nama_container> <port> <api_url> <token>"
    echo "  $0 deprovision <nama_container>"
    echo "  $0 reset <nama_container>"
    exit 1
}

ACTION=$1

case "$ACTION" in

    provision)
        MASTER_CONTAINER=$2
        PROFILE=$3
        NAME=$4
        PORT=$5
        API_URL=$6
        TOKEN=$7

        if [ -z "$TOKEN" ]; then
            echo "Error: token tidak boleh kosong"
            usage
        fi

        if ! lxc info "$MASTER_CONTAINER" > /dev/null 2>&1; then
            echo "Error: master container '$MASTER_CONTAINER' tidak ditemukan"
            exit 1
        fi
        if ! lxc profile show "$PROFILE" > /dev/null 2>&1; then
            echo "Error: profile '$PROFILE' tidak ditemukan"
            exit 1
        fi
        if lxc info "$NAME" > /dev/null 2>&1; then
            echo "Error: container '$NAME' sudah ada di LXD"
            exit 1
        fi

        lxc copy "$MASTER_CONTAINER" "$NAME" --storage pool-lab --profile "$PROFILE" || exit 1

        lxc config set "$NAME" environment.PRAKTIKUM_API_URL "$API_URL"
        lxc config set "$NAME" environment.PRAKTIKUM_API_TOKEN "$TOKEN"

        lxc start "$NAME" || exit 1
        lxc config device add "$NAME" proxy22 proxy listen=tcp:0.0.0.0:"$PORT" connect=tcp:127.0.0.1:22

        for attempt in $(seq 1 10); do
            STATE=$(lxc info "$NAME" | grep "^Status:" | awk '{print $2}')
            if [ "$STATE" == "Running" ]; then
                break
            fi
            sleep 1
        done

        lxc snapshot "$NAME" clean
        echo "OK"
        ;;

    deprovision)
        NAME=$2
        if [ -z "$NAME" ]; then
            echo "Error: nama container wajib diisi"
            usage
        fi
        if lxc info "$NAME" > /dev/null 2>&1; then
            lxc stop "$NAME" --force
            lxc delete "$NAME"
        fi
        echo "OK"
        ;;

    reset)
        NAME=$2
        if [ -z "$NAME" ]; then
            echo "Error: nama container wajib diisi"
            usage
        fi
        if ! lxc info "$NAME" > /dev/null 2>&1; then
            echo "Error: container '$NAME' tidak ditemukan"
            exit 1
        fi
        if ! lxc info "$NAME" | grep -q "clean"; then
            echo "Error: snapshot 'clean' tidak ditemukan di '$NAME'"
            exit 1
        fi
        lxc restore "$NAME" clean
        echo "OK"
        ;;

    *)
        usage
        ;;
esac
