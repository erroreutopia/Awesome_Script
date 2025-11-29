#!/bin/env python3
import os
import requests
from bs4 import BeautifulSoup
def download_image(url, folder='images'):   
    if not os.path.exists(folder):
        os.makedirs(folder)
    response = requests.get(url)
    filename = url.split("/")[-1]
    filepath = os.path.join(folder, filename)
    with open(filepath, 'wb') as f:
        f.write(response.content)
    print(f"图片已保存至 {filepath}")
def get_images(url):
    response = requests.get(url)
    soup = BeautifulSoup(response.text, 'html.parser')
    images = []
    for img in soup.find_all('img'):
        img_url = img.get('src')
        if not img_url:
            continue
        if not img_url.startswith(('http://', 'https://')):
            base_url = url.split('/')[0] + '//' + url.split('/')[2]
            img_url = base_url + '/' + img_url.lstrip('/')
        images.append(img_url)
    return images
def main():
    target_url = input("请输入要爬取图片的网址: ")
    images = get_images(target_url)
    for i, image in enumerate(images):
        print(f"正在下载第 {i+1} 张图片...")
        download_image(image)
    print("所有图片已下载完成。")
if __name__ == "__main__":
    main()
