if [ -f .env ]; then
  export $(cat .env | xargs)
else
  echo 'Missing .env file, see .env.example for reference'
  exit 1
fi

curl -X POST ${APP_URL}slack_hook -d "text=\!$*&trigger_word=\!&user_name=fcoury&channel_name=other"
