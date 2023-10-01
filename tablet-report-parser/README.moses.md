<a name="readme-top"></a>

<!-- PROJECT SHIELDS -->

<h1 align="center">Moses.pl</h1>

<div align="center">
  <a href="https://github.com/yugabyte/yb-tools/tree/main/tablet-report-parser">
    <img src="moses-dropping-a-tablet.png" alt="Logo" >
  </a>

  <h3 align="center">
    Fetches and analyzes tablets for a Universe</h3>
    <p/>
    <a href="https://github.com/yugabyte/yb-tools/tree/main/tablet-report-parser"><strong>Explore the docs »</strong></a>
    <a href="https://github.com/yugabyte/yb-tools/tree/main/tablet-report-parser">View Demo</a>
    ·
    <a href="https://github.com/yugabyte/yb-tools/tree/main/tablet-report-parser/issues">Report Bug</a>
    ·
    <a href="https://github.com/yugabyte/yb-tools/tree/main/tablet-report-parser/issues">Request Feature</a>
</div>



<!-- TABLE OF CONTENTS -->
<details>
  <summary>Table of Contents</summary>
  <ol>
    <li>
      <a href="#about-the-project">About The Project</a>
      <ul>
        <li><a href="#built-with">Built With</a></li>
      </ul>
    </li>
    <li>
      <a href="#getting-started">Getting Started</a>
      <ul>
        <li><a href="#prerequisites">Prerequisites</a></li>
        <li><a href="#installation">Installation</a></li>
      </ul>
    </li>
    <li><a href="#usage">Usage</a></li>
    <li><a href="#contact">Contact</a></li>

  </ol>
</details>



<!-- ABOUT THE PROJECT -->
## About The Project

Moses.pl is intended to replace the current process of obtaining and analyzing a tablet report (yugatool/tablet-report-parser).

<p align="right">(<a href="#readme-top">back to top</a>)</p>


<!-- GETTING STARTED -->
## Getting Started

To get a local copy up and running follow these simple example steps:

Download and install the code on any linux host that has access to the YBA host.

You will need the YBA access token, and the name of the uiverse whose tablets you want to analyze.

### Prerequisites

The host must have perl >= 5.16 installed.

### Installation

Download the code to a suitable directory, and make it executable.

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- USAGE EXAMPLES -->
## Usage
 
 `perl moses.pl  --YBA_HOST=https://Your-yba-hostname-or-IP --API_TOKEN=Your-token  --univ Your-universe-name`

<p align="right">(<a href="#readme-top">back to top</a>)</p>



<!-- CONTACT -->
## Contact

<a href="https://github.com/na6vj">NA6VJ</a>

Project Link: [https://github.com/na6vj/yb-tools](https://github.com/na6vj/yb-tools)

<p align="right">(<a href="#readme-top">back to top</a>)</p>

