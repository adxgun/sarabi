---
title: Introduction
description: Definition and description of sarabi
---

## What is Sarabi?
`sarabi` is a full-stack application deployment tool designed with simplicity and security in mind. It aims to streamline the deployment process for small-scale applications by requiring minimal configuration. With sarabi, multiple applications, including static websites, can be managed on a single server effortlessly. 

Tools like Kubernetes are often too complex for simple use cases, and also they demand significant expertise. sarabi provides a lightweight, straightforward solution for managing deployments and operations for smaller-scale projects.

Sarabi is a full automation of my personal workflow which is very efficient and I've found it very useful over the years, and I hope you will have the same experience.

## What makes sarabi different?
* **Full-Stack Deployment Made Easy:** Unlike many PaaS tools that require separate solutions for backend and static frontend hosting, sarabi integrates both. It uses a single Caddy instance to serve your frontend and backend with automatic HTTPS.
* **Unlimited Environments:** Create as many environments as you need (e.g., dev, prod, pr-123) without maintaining separate configuration files. Each environment has unique access URLs, databases, and configurations etc.
* **Enhanced Database Security:** By default, databases are not exposed to the internet. Only the associated application can connect to the database, with an option to allow specific IPs access.
* **Customizable Database Backups:** sarabi creates automatic database backups every 8 hours by default. You can adjust the schedule to your needs—hourly, every 15 minutes, or any other interval. Backups can be restored easily when required.
* **Specific Application Deployment Template:** This template is specifically designed for managing 3-tier applications. Any application that requires one or more databases (PostgreSQL, MySQL, Redis, or MongoDB) along with a single runner binary is ideally suited for Sarabi to handle.

## Use cases
* Small to medium traffic web applications
* Internal services
* Application development agencies seeking a smarter way to manage client deployments
* Solo entrepreneurs aiming to optimize costs by staying lean