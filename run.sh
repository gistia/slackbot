if [ -f .env ]; then
  export $(cat .env | xargs)
else
  echo 'Missing .env file, see .env.example for reference'
  exit 1
fi
gin
