# Subdomain Crawler

The program aims to help you collect subdomains of a list of given second-level domains (SLD).

![](./assets/demo.gif)

## Usage

1. Edit input file `input.txt`

```bash
$ head input.txt
tsinghua.edu.cn
pku.edu.cn
fudan.edu.cn
sjtu.edu.cn
zju.edu.cn
```
2. Run the program

```bash
$ ./subdomain-crawler -n 64
```
3. Check out the result in `output/` folder.

```bash
$ head output/*
```
