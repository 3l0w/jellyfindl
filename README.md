# JellyfinDL

A tui for downloading your Jellyfin content into your computer


## :package: Installation

* Grab the executable in the [release](https://github.com/3l0w/jellyfindl/releases) page

* Or use the go install command

```bash
go install github.com/3l0w/jellyfindl
```

## :camera_flash: Screenshots

![Home page](https://i.imgur.com/S9JjsVw.png)

![Download page](https://i.imgur.com/Z9p9BXD.png)

## :question: Usage

On the first launch you will be asked: 

*  The endpoint of your Jellyfin instance, example: `https://jellyfin.example.com`
* An API key that you can generate inside the admin dashboard on the API keys section
* Your UserId you can find it when opening your profile in the URL
  ![image-20221117154803403](https://i.imgur.com/MNsBEEJ.png)

To select your medias use the `arrow` keys to move and press `enter` to select

To start downloading press the `tab` button that will set you on the bottom button and simply press `enter`.

To delete a file that you have downloaded hover it and press `d`

Quit the program using `q` or `ctrl+c`

## :gear: Building

You need at least go 18 installed (i use personally go 19)

```bash
go build
```