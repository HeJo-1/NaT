
# NaT OSINT Tool

<img width="1024" height="1024" alt="logo" src="https://github.com/user-attachments/assets/ad219fb0-d1d9-4fae-afb9-b78b43c37add" />


This is a command-line tool written in Go to automate various Open Source Intelligence (OSINT) tasks. It features four main modules:

1.  **Username Search:** Searches for a given username across more than 30 popular social media and web platforms.
2.  **Website Text Similarity:** Compares the text content of given URLs and calculates the Jaccard similarity score between them.
3.  **Reverse Image Search:** Performs a search using Google Lens with a specified image and lists potentially related web pages.
4.  **Geolocation from Image:** Analyzes the EXIF metadata of an image file to extract geographic location (latitude, longitude) if available, and finds the corresponding address.

## Features

-   **Fast and Concurrent:** The username search module operates quickly using concurrent requests, configurable via the `concurrency` parameter.
-   **Wide Site Support:** Checks for usernames on popular platforms like Facebook, Twitter, Instagram, GitHub, Reddit, and many more.
-   **Alternative Search:** Provides an option to perform an additional search by inverting the case of the username's characters.
-   **JSON Output:** Saves search results to a `.json` file for easy processing.
-   **Modular Design:** Easily run the desired function using the `mode` parameter.
-   **Leverages External Libraries:** Utilizes powerful libraries such as `goquery`, `goexif2`, and more.

## Installation

To use this tool, you must have the **Go** programming language installed on your system.

1.  **Install Go:**
    If you don't have Go installed, download and install it from the [official Go website](https://go.dev/doc/install).

2.  **Clone the Project:**
    Open a terminal or command prompt and run the following command to download the project to your computer:
    ```sh
    git clone https://github.com/HeJo-1/NaT/
    cd NaT
    ```

3.  **Install Dependencies:**
    While in the project directory, run the following command to download the required external Go modules. This command will automatically analyze the `go.mod` file and fetch all dependencies.
    ```sh
    go mod tidy
    ```

4.  **Build the Project (Optional):**
    Instead of running the project with `go run NaT.go` every time, you can create a single executable file.
    ```sh
    go build
    ```
    This command will create an executable file named `NaT` (on Linux/macOS) or `NaT.exe` (on Windows) in the project directory.

## Usage

The tool is controlled via the `-mode` flag. Each mode has its own specific additional flags.

If you are using the compiled executable, replace `go run NaT.go` with `./NaT` (Linux/macOS) or `NaT.exe` (Windows).

---

### Module 1: Username Search

Searches for a specified username across social media platforms.

**Parameters:**

-   `-mode=username` (Required)
-   `-username="<username>"` (Required): The username to search for.
-   `-o="<filename>.json"` (Optional): The file to save the results to. Default: `results.json`.
-   `-c=<number>` (Optional): The number of concurrent requests. Default: `6`.
-   `-t=<seconds>` (Optional): The timeout duration for each request. Default: `10`.
-   `-a` (Optional): A flag to perform an additional search by inverting the case of the username (e.g., `Example` -> `eXAMPLE`).

**Examples:**

-   Basic search:
    ```sh
    go run NaT.go -mode=username -username="johndoe"
    ```

-   Alternative search, saving the output to `doe_results.json`:
    ```sh
    go run NaT.go -mode=username -username="JohnDoe" -a -o="doe_results.json"
    ```

---

### Module 2: Website Text Similarity

Compares the text content of websites from a comma-separated list of URLs.

**Parameters:**

-   `-mode=websimilarity` (Required)
-   `-urls="<url1>,<url2>,<url3>"` (Required): The URLs of the websites to compare.

**Example:**

```sh
go run NaT.go -mode=websimilarity -urls="https://www.gutenberg.org/files/1342/1342-h/1342-h.htm,https://www.gutenberg.org/files/98/98-h/98-h.htm"
```

---

### Module 3: Reverse Image Search (Google Lens)

Searches for a given image file using Google Lens.

**Parameters:**

-   `-mode=lens` (Required)
-   `-image="<filepath>"` (Required): The file path to the image to be searched.

**Example:**

```sh
go run NaT.go -mode=lens -image="./images/test_photo.jpg"
```

> **Warning:** This module works by web scraping Google's interface. If Google changes its HTML structure, this feature may not function correctly.

---

### Module 4: Geolocation from Image

Extracts GPS coordinates from an image file's EXIF data.

**Parameters:**

-   `-mode=geo` (Required)
-   `-image="<filepath>"` (Required): The file path of the image to be analyzed.

**Example:**

```sh
go run NaT.go -mode=geo -image="./images/photo_with_gps.jpg"
```

> **Note:** For this module to work, the photo must contain embedded GPS data (latitude/longitude). Most modern smartphones add this data to photos when location services are enabled.

## Dependencies

This project uses the following external Go libraries:

-   [github.com/PuerkitoBio/goquery](https://github.com/PuerkitoBio/goquery)
-   [github.com/codingsince1985/geo-golang/openstreetmap](https://github.com/codingsince1985/geo-golang)
-   [github.com/cozy/goexif2/exif](https://github.com/cozy/goexif2)
-   [github.com/gookit/color](https://github.com/gookit/color)
