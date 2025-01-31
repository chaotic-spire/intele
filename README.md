# Intele

[![Go Reference](https://pkg.go.dev/badge/github.com/nlypage/intele.svg)](https://pkg.go.dev/github.com/nlypage/intele)
[![Go Report Card](https://goreportcard.com/badge/github.com/nlypage/intele)](https://goreportcard.com/report/github.com/nlypage/intele)
[![License](https://img.shields.io/github/license/nlypage/intele)](LICENSE)

Intele is a powerful and flexible input management library for Telegram bots built
with [telebot.v3](https://github.com/tucnak/telebot). It provides a simple and efficient way to handle user input
requests and button callbacks in your Telegram bot applications.

## Features

- 🚀 Easy-to-use input request management
- 🎮 Support for button callbacks
- ⏱️ Configurable timeout support
- 🔄 Context-aware operations
- 💾 Flexible state storage system
- 🛡️ Thread-safe implementation
- 🎯 Cancellable input requests
- 🧹 Message collector for clean input handling

## Installation

```bash
go get github.com/nlypage/intele
```

## Quick Start

Here's a simple example of how to use Intele in your Telegram bot:

```go
package main

import (
  "context"
  "github.com/nlypage/intele"
  tele "gopkg.in/telebot.v3"
  "log"
  "time"
)

func main() {
  // Initialize your bot
  b, err := tele.NewBot(tele.Settings{
    Token:  "YOUR_BOT_TOKEN",
    Poller: &tele.LongPoller{Timeout: 10 * time.Second},
  })
  if err != nil {
    log.Fatal(err)
  }

  // Create input manager
  inputManager := intele.NewInputManager(intele.InputOptions{})

  // Set up the input handlers
  b.Handle(tele.OnText, inputManager.MessageHandler())
  b.Handle(tele.OnCallback, inputManager.CallbackHandler())

  // Example command that waits for user input or button press
  b.Handle("/ask", func(c tele.Context) error {
    // Create buttons
    btn := &tele.Btn{
      Text:   "Click me",
      Unique: "btn1",
    }

    // Send message with buttons
    _ = c.Send("Please enter text or click a button:", &tele.ReplyMarkup{
      InlineKeyboard: [][]tele.InlineButton{{*btn.Inline()}},
    })

    // Wait for user input (with 1 minute timeout)
    response, err := inputManager.Get(
      context.Background(),
      c.Sender().ID,
      time.Minute,
      btn,
    )

    if err != nil {
      return c.Send("Error getting input: " + err.Error())
    }
    if response.Canceled {
      return c.Send("Input canceled")
    }

    // Handle text message
    if response.Message != nil {
      return c.Send("You entered: " + response.Message.Text)
    }

    // Handle button callback
    if response.Callback != nil {
      return c.Send("You clicked the button!")
    }

    return nil
  })

  b.Start()
}
```

## Documentation

### Creating an Input Manager

```go
// Create with default memory storage
inputManager := intele.NewInputManager(intele.InputOptions{})

// Create with custom storage
inputManager := intele.NewInputManager(intele.InputOptions{
Storage: myCustomStorage,
})
```

### Handling Input

The library supports two types of input:

1. Text messages - handled by `MessageHandler()`
2. Button callbacks - handled by `CallbackHandler()`

Make sure to register both handlers with your bot:

```go
b.Handle(tele.OnText, inputManager.MessageHandler())
b.Handle(tele.OnCallback, inputManager.CallbackHandler())
```

When using `Get()`, you can pass button endpoints to handle callbacks:

```go
btn := &tele.Btn{Text: "Click me", Unique: "btn1"}
response, err := inputManager.Get(ctx, userID, timeout, btn)

if response.Message != nil {
// Handle text message
fmt.Println("Text:", response.Message.Text)
} else if response.Callback != nil {
// Handle button callback
fmt.Println("Button clicked:", response.Callback.Unique)
}
```

### Message Collector

The library includes a message collector that helps manage and clean up messages during input operations. This is
especially useful when you need to collect and later remove all messages that were part of an input sequence.

```go
// Example of using collector in an input loop
func handleUserInput(c tele.Context) error {
    inputCollector := collector.New()
    
    // Send initial message and collect it
    _ = inputCollector.Send(c,
        "Please enter your full name:",
        &tele.ReplyMarkup{...},
    )

    for {
        // Wait for user input
        message, canceled, err := inputManager.Get(context.Background(), c.Sender().ID, 0)
        if message != nil {
            inputCollector.Collect(message) // Collect user's message
        }

        switch {
        case canceled:
            // Clear all messages except the last one
            _ = inputCollector.Clear(c, collector.ClearOptions{
                IgnoreErrors: true,
                ExcludeLast: true,
            })
            return nil
        case err != nil:
            // Send error message and collect it
            _ = inputCollector.Send(c,
                "Error occurred. Please try again.",
                &tele.ReplyMarkup{...},
            )
        case isValidInput(message.Text):
            // Clear all collected messages
            _ = inputCollector.Clear(c, collector.ClearOptions{
                IgnoreErrors: true,
            })
            return processInput(message.Text)
        default:
            // Send invalid input message and collect it
            _ = inputCollector.Send(c,
                "Invalid input. Please try again.",
                &tele.ReplyMarkup{...},
            )
        }
    }
}
```

The collector provides the following methods:

- `New()` - Creates a new message collector
- `Collect(message)` - Adds a message to the collector
- `Send(context, what, opts...)` - Sends a message and automatically collects it
- `Clear(context, options)` - Deletes all collected messages with configurable options:
  - `IgnoreErrors` - Continue deletion even if some messages fail to delete
  - `ExcludeLast` - Keep the last collected message when clearing

### Key Methods

- `NewInputManager(options)` - Creates a new input manager with the given options
- `MessageHandler()` - Returns a handler for text messages that should be registered with `tele.OnText`
- `CallbackHandler()` - Returns a handler for button callbacks that should be registered with `tele.OnCallback`
- `Get(ctx, userID, timeout, callbacks...)` - Waits for user input or button press. Returns a Response containing either
  Message or Callback
- `Cancel(userID)` - Cancels any pending input request for the given user
- `Collector.New()` - Creates a new message collector
- `Collector.Collect(message)` - Adds a message to the collector
- `Collector.Send(context, what, opts...)` - Sends a message and automatically collects it
- `Collector.Clear(context, options)` - Clears collected messages with the given options

### Error Handling

The library provides proper error handling for various scenarios:

- Context cancellation
- Timeout errors
- State storage errors

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

If you encounter any problems or have questions, please [open an issue](https://github.com/nlypage/intele/issues/new).

---
Made with ❤️ by [nlypage](https://github.com/nlypage)
