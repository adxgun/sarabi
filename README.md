### Floki

Floki is a full-stack application deployment tool designed with simplicity and security in mind. It aims to streamline the deployment process for small-scale applications by requiring minimal configuration. With Floki, multiple applications, including static websites, can be managed on a single server effortlessly. Unlike complex tools like Kubernetes that demand significant expertise, Floki provides a lightweight, straightforward solution for managing deployments and operations for smaller-scale projects.

## Who Can Use Floki
Floki is ideal for:
- Small to medium traffic web applications
- Internal services
- Application development agencies seeking a smarter way to manage client deployments
- Solo entrepreneurs aiming to optimize costs by staying lean

## Key Features
- **Language Agnostic:** Deploy web applications written in any programming language with just a `Dockerfile`.
- **Full-Stack Deployment:** Supports deploying both service APIs and frontend (static files) in a single step.
- **Multi-Environment Support:** Easily spin up multiple replicas of your app in different environments (e.g., `dev`, `prod`) with a single command.
- **Database Resource Provisioning:** Built-in support for PostgreSQL, MySQL, MongoDB, and Redis.
- **Database Access Restriction:** Restricts database connections from the outside world by default to enhance security. You can whitelist or blacklist IPs based on your preferences.
- **Automatic Database Backups:** Automatically backs up your database when you deploy an app with a database. Backups are stored on the server or can be configured to use S3-compatible object storage (recommended).
- **Automatic HTTPS:** Powered by Caddy for seamless HTTPS setup.
- **Multi-Application Management:** Deploy and manage multiple applications on a single server.
- **Logs Management:** Access application logs with customizable log retention timelines.
- **Scalability:** Deploy multiple replicas of your application to achieve horizontal scalability.
- **Rollback Support:** Quickly rollback to any previous version of your application using a unique identifier if needed.

## How Floki Stands Out from Existing PaaS Tools
- **Full-Stack Deployment Made Easy:** Unlike many PaaS tools that require separate solutions for backend and static frontend hosting, Floki integrates both. It uses a single Caddy instance to serve your frontend and backend with automatic HTTPS.
- **Unlimited Environments:** Create as many environments as you need (e.g., `dev`, `prod`, `pr-123`) without maintaining separate configuration files. Each environment has unique access URLs, databases, and configurations.
- **Enhanced Database Security:** By default, databases are not exposed to the internet. Only the associated application can connect to the database, with an option to allow specific IPs access.
- **Customizable Database Backups:** Floki creates automatic database backups every 8 hours by default. You can adjust the schedule to your needs—hourly, every 15 minutes, or any other interval. Backups can be restored easily when required.

## Use Cases
Floki is perfect for developers, startups, and small teams that need a reliable yet straightforward deployment solution. By minimizing complexity and maximizing functionality, Floki ensures you can focus on building great applications without being bogged down by operational overhead.

## Getting Started