# WFH Calendar CLI

A simple Go CLI to quickly register a "Work From Home" event in your Google Calendar.

## Description

This CLI tool is designed for developers who want a quick way to mark a day as "Work From Home" in their Google Calendar. It fetches the `$USER` environment variable for the user's name and creates an all-day event titled "$USER wfh" in a predefined Google Calendar.

## Features

- Integrates with Google Calendar API.
- Stores configuration in `~/.wfh`, making it accessible across the machine.
- Supports OAuth2 for authentication.
- Automatically saves and reuses authentication tokens.

## Prerequisites

- Go (v1.15 or later)
- A Google account with Calendar API access.
- `credentials.json` obtained from the [Google Developer Console](https://console.developers.google.com/).

## Setup

1. Install the CLI tool:
   ```bash
   go install https://github.com/perbu/wfh.git
   ```

2. Place your `credentials.json` in the `~/.wfh` directory.


## Usage

1. Run the CLI for the first time:
   ```bash
   wfh
   ```
   On the first run, you'll be prompted to authorize the application to access your Google Calendar. Follow the link provided, authorize the application, and paste the code back into the terminal.

2. To mark today as a WFH day:
   ```bash
   .wfh
   ```

3. Check Google Calendar. You should see a new all-day event titled "$USER wfh".

## Configuration

The tool will generate a configuration file in `~/.wfh/config.json`. You can modify this file to change:

- `CalendarID`: The ID of your Google Calendar (default is "primary").
- `DefaultMessage`: The default message for the event (default is "wfh").
- `User`: If you wish to override the `$USER` environment variable, specify the name here.

## Contributions

Feel free to open an issue or submit a pull request if you have suggestions, improvements, or bug fixes. All contributions are welcome!

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE.md) for details.
