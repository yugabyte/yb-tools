#!/bin/bash

# Usage: ./find_zombies.sh <master_ip:port> <database> <table_name> [--check-backends]
# Example: ./find_zombies.sh yb-my-cluster-node1:7000 yugabyte users --check-backends

MASTER="$1"
DATABASE="$2"
TABLE="$3"
CHECK_BACKENDS="$4"
TSERVER_HTTP_PORT=9000
YSQL_PORT=5433

if [[ -z "$MASTER" || -z "$DATABASE" || -z "$TABLE" ]]; then
  echo "Usage: $0 <master_ip:port> <database> <table_name> [--check-backends]"
  echo ""
  echo "  --check-backends  Auto-discovers all TServers and checks pg_stat_activity"
  echo "                    on every node to verify if a blocker is live or a zombie."
  exit 1
fi

TSERVER_HOSTS=""

if [[ -n "$CHECK_BACKENDS" ]]; then
  TSERVER_HOSTS=$(yb-admin --master_addresses "$MASTER" list_all_tablet_servers 2>/dev/null \
    | grep -oP '\S+(?=:9100)' | sort -u)

  if [[ -z "$TSERVER_HOSTS" ]]; then
    echo "WARNING: Could not discover TServer hosts. Backend check disabled."
    CHECK_BACKENDS=""
  else
    reachable=false
    for host in $TSERVER_HOSTS; do
      if ysqlsh -h "$host" -p "$YSQL_PORT" -d "$DATABASE" -t -A -c "SELECT 1;" >/dev/null 2>&1; then
        reachable=true
        break
      fi
    done
    if [[ "$reachable" == "false" ]]; then
      echo "WARNING: Cannot connect to any TServer via ysqlsh (port $YSQL_PORT). Backend check disabled."
      CHECK_BACKENDS=""
    fi
  fi
fi

check_backend() {
  local txn_id="$1"
  for host in $TSERVER_HOSTS; do
    result=$(ysqlsh -h "$host" -p "$YSQL_PORT" -d "$DATABASE" -t -A -c \
      "SELECT pid || '|' || state || '|' || substr(query,1,80) FROM pg_stat_activity WHERE yb_backend_xid = '${txn_id}';" 2>/dev/null)
    if [[ -n "$result" ]]; then
      echo "${host}|${result}"
      return 0
    fi
  done
  return 1
}

echo "============================================================"
echo " Zombie Transaction Scanner"
echo " Table: ${DATABASE}.${TABLE}"
echo " Master: ${MASTER}"
if [[ -n "$CHECK_BACKENDS" ]]; then
  echo " TServers: $(echo $TSERVER_HOSTS | tr '\n' ' ')"
fi
echo "============================================================"
echo ""

tablets_tmp=$(mktemp)
trap "rm -f $tablets_tmp" EXIT

yb-admin --master_addresses "$MASTER" list_tablets "ysql.${DATABASE}" "$TABLE" 0 2>/dev/null > "$tablets_tmp"

found_any=false

while IFS= read -r line; do
  tablet_id=$(echo "$line" | grep -oP '[a-f0-9]{32}' | head -1)
  leader_host=$(echo "$line" | grep -oP '\S+(?=:9100)' | tail -1)

  [[ -z "$tablet_id" || -z "$leader_host" ]] && continue

  wq_html=$(curl -s "http://${leader_host}:${TSERVER_HTTP_PORT}/waitqueue?id=${tablet_id}")

  blocker_ids=$(echo "$wq_html" \
    | sed -n '/Blockers:/,/<\/table>/p' \
    | grep -oP '[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}' \
    | sort -u)

  txn_html=$(curl -s "http://${leader_host}:${TSERVER_HTTP_PORT}/transactions?id=${tablet_id}")
  pending_count=$(echo "$txn_html" | grep -ic "PENDING")

  [[ -z "$blocker_ids" && "$pending_count" -eq 0 ]] && continue

  found_any=true
  echo "------------------------------------------------------------"
  echo " Tablet:  $tablet_id"
  echo " Leader:  $leader_host"
  echo "------------------------------------------------------------"

  if [[ -n "$blocker_ids" ]]; then
    echo ""
    echo " Blocker Transactions (from /waitqueue):"
    echo "$blocker_ids" | while IFS= read -r txn_id; do
      [[ -z "$txn_id" ]] && continue
      status=$(echo "$wq_html" \
        | sed -n '/Blockers:/,/<\/table>/p' \
        | grep -A1 "$txn_id" \
        | grep -oP '(PENDING|released|committed|aborted)' \
        | head -1)
      status=${status:-UNKNOWN}
      echo "   txn: $txn_id  status: $status"

      if [[ -n "$CHECK_BACKENDS" ]]; then
        backend_result=$(check_backend "$txn_id")
        if [[ $? -eq 0 ]]; then
          be_host=$(echo "$backend_result" | cut -d'|' -f1)
          pid=$(echo "$backend_result" | cut -d'|' -f2)
          state=$(echo "$backend_result" | cut -d'|' -f3)
          query=$(echo "$backend_result" | cut -d'|' -f4)
          echo "   verdict: LIVE on $be_host (pid=$pid, state=$state)"
          echo "   query:   $query"
        else
          echo "   verdict: ** ZOMBIE ** (no backend found on any TServer)"
          echo "   action:  SELECT yb_cancel_transaction('${txn_id}'::uuid);"
        fi
      fi
    done
  fi

  if [[ "$pending_count" -gt 0 ]]; then
    echo ""
    echo " PENDING transactions on /transactions page: $pending_count"
    echo " Review: http://${leader_host}:${TSERVER_HTTP_PORT}/transactions?id=${tablet_id}"
  fi

  echo ""
done < "$tablets_tmp"

if [[ "$found_any" == "false" ]]; then
  echo "No blockers or pending transactions found. Table looks clean."
fi
