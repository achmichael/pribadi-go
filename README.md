# Building a Privacy-First Personal AI Assistant in Go

## Overview

I built `pribadi-go` as a personal AI Assistant project with one main goal: to understand how an AI Assistant can be built from the ground up while keeping user data as private as possible.

The project is based on a privacy-first approach, where personal information should remain within the user's own environment whenever possible. Instead of treating an LLM as the entire application, I designed the system around the LLM by adding components for memory, data storage, document processing, retrieval, and application logic.

The project is written in Go and uses components such as SQLite for local persistence and Qdrant for vector search. The project also includes document-processing dependencies for working with files such as PDF and DOCX documents.

## The Problem

Modern AI assistants are becoming increasingly useful for personal and professional tasks. However, using an AI service often means sending conversation history, documents, or other personal information to an external service.

This raised an engineering question for me:

> Can I build a personal AI Assistant that provides useful assistant-like capabilities while keeping the user's data under their control?

The goal was not to recreate ChatGPT or Claude completely. Instead, I wanted to understand which components are required to build an AI Assistant and how those components can be designed around privacy.

## System Architecture

I separated the application into several responsibilities instead of putting all logic into a single chatbot handler.

The project contains components for:

* Identity and user-related information
* Fact and long-term memory
* Document chunking
* Repository and persistence logic
* Application use cases
* Delivery and communication layers

This separation makes it easier to evolve individual components without tightly coupling the entire system.

For example, memory should not be responsible for handling HTTP requests, while the delivery layer should not need to understand how vector search is implemented.

This approach also makes the project easier to extend because new capabilities can be added as separate use cases or components.

## Memory and Retrieval

One of the most important parts of the project is memory.

A conversational AI should not treat every message as an isolated request. It needs to distinguish between information that is relevant only to the current conversation and information that may be useful in future interactions.

For long-term information, I use a retrieval-based approach. Information can be processed, stored, and later retrieved when the assistant needs additional context.

Vector search is useful for this because a user's query does not always contain the exact words that were used when the information was originally stored.

For example, a user might previously mention:

> "I usually work on my thesis at night."

Later, they might ask:

> "When do I normally work on my research?"

A semantic retrieval system can help connect these two pieces of information even though the wording is different.

## Document Processing

Another requirement for a personal assistant is the ability to work with user-owned documents.

Instead of sending an entire document directly into a model every time it is needed, the document can be processed into smaller chunks.

A simplified pipeline looks like this:

```text
Document
   ↓
Text Extraction
   ↓
Chunking
   ↓
Embedding
   ↓
Vector Storage
   ↓
Semantic Retrieval
   ↓
Relevant Context
   ↓
LLM
```

Chunking is important because documents can be much larger than the context that should be provided to the model for a single request.

Retrieval allows the assistant to provide only the relevant parts of a document instead of loading the entire document into every interaction.

## Local-First Data Storage

Privacy was one of the main reasons I chose to build this project.

The application uses local persistence for application data, while vector data can be handled through a locally managed vector database.

This architecture allows the system to be deployed within an environment controlled by the user rather than requiring all application data to be stored by a third-party SaaS platform.

However, I consider "zero data exposure" as an architectural goal rather than a claim that automatically applies to every deployment.

If an external model or API is used, the data flow depends on that provider and configuration. A fully local deployment with a local model provides a different privacy boundary from a deployment that communicates with an external LLM API.

## Engineering Challenges

The most difficult part of this project was not connecting an LLM to an application.

The harder problem was designing the system around the LLM.

For example, the assistant needs to decide:

1. What information from the current conversation is relevant?
2. What information should be retrieved from long-term memory?
3. What information should be stored for future conversations?
4. How should documents be processed before retrieval?
5. How should retrieved information be combined with the current context?
6. How can the system remain modular as new capabilities are added?

These questions pushed me to think about the AI Assistant as a software system rather than simply an interface to an LLM.

## Why I Built It

The main reason I built `pribadi-go` was to learn.

I wanted to move beyond simply using AI APIs and understand the engineering behind an AI Assistant: memory management, retrieval, persistence, context handling, document processing, and system architecture.

At the same time, I wanted to explore a problem that I believe will become increasingly important as AI becomes part of everyday life: how to obtain the benefits of AI without unnecessarily giving up control over personal data.

The project is still evolving, and I do not consider it equivalent to large commercial AI assistants. The value of the project for me is the engineering experience it provides and the opportunity to experiment with how a privacy-first AI Assistant can be designed.
