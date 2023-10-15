# WFH Calendar CLI

A simple Go CLI to quickly register a "Work From Home" event in your Google Calendar.

## Description

This CLI tool is designed for developers who want a quick way to mark a day as "Work From Home" 
in their Google Calendar.


## Features

- Integrates with Google Calendar API.
- Stores configuration in `~/.wfh`, making it accessible across the machine.
- Supports OAuth2 for authentication.
- Automatically saves and reuses authentication tokens.
- Allows embedding of Google credentials in the binary for simpler distribution.
- Works on Windows, macOS, and Linux.

## Prerequisites

- Go (v1.21 or later)
- A Google account with Calendar API access.
- `credentials.json` obtained from the [Google Developer Console](https://console.developers.google.com/).

You can skip DefaultMessage and User if you want to use the defaults. The default for User is to use $USER.

## Usage

1. Run the CLI for the first time, we use -list not to create a new event but for auth/authz.
   ```bash
   wfh -list
   ```
   On the first run, you'll be prompted to authorize the application to access your Google Calendar. 

2. To mark today as a WFH day:
   ```bash
   wfh [--date 2023-03-01] <optional message>
   ```
3. Check Google Calendar. You should see a new all-day event titled with your default message.

## Contributions

Feel free to open an issue or submit a pull request if you have suggestions, improvements, or bug fixes. 
All contributions are welcome! 

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE.md) for details.
