![KTP Logo](https://media.licdn.com/dms/image/v2/D4E0BAQGrPXnp3Q6Lag/company-logo_200_200/company-logo_200_200/0/1736456274837/kappa_theta_pi_uga_logo?e=2147483647&v=beta&t=b4XekMawqkH_qvNb2kgbn87eem6Gn78cfDFWixzLY6U)

# Internly

A Discord bot that searches for internships and new grad positions and posts them to a specified Discord channel.

## Features

-   Setup Discord channels to receive internships and new grad positions and post them 
-   Subscribe to personalized notifications for specific internships
-   Filter by location, role, company, and job type

## Installation

Use the Dockerfile and the Docker Compose script to spin up a new instance easily.

First, create a `config.json` file and follow the template:

```json
  {
    "discordToken": "<token>",
    "pollTime": "1h"
}
```

Then run: `docker compose up -d`

## Commands

- `/configure` - Configure the Discord channels to receive job postings.
- `/subscriptions` - View your personal subscriptions
- `/subscribe` - Set up a new subscription
- `/unsubscribe` - Stop receiving notifications for a specified subscription
- `/help` - View a help menu

## Badges

[![MIT License](https://img.shields.io/badge/License-MIT-green.svg)](https://github.com/StephenSulimani/internly-bot/blob/master/LICENSE)
[![Go](https://img.shields.io/badge/Go-00ADD8?logo=Go&logoColor=fff)](#)
[![Docker](https://img.shields.io/badge/Docker-1D63ED?logo=Docker&logoColor=fff)](#)
[![TypeScript](https://img.shields.io/badge/Invite%20Me-7289da?logo=Discord&logoColor=fff)](https://discord.com/oauth2/authorize?client_id=1401393902203699370&scope=bot&permissions=377957125184)
[![GroupMeBot](https://img.shields.io/github/stars/StephenSulimani/internly-bot)](https://github.com/StephenSulimani/GroupMeBot)

## Contributing

Contributions are always welcome!

Please try to maintain the current code style.

## License

[MIT](https://github.com/StephenSulimani/GroupMeBot/blob/master/LICENSE)
