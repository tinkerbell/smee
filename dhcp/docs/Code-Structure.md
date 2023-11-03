# Code Structure

## Backend

Responsible for communicating with an external persistence source and returning data from said source.
Backends live in the `backend/` directory.

## Handler

Responsible for reading a DHCP packet from a source, calling a backend, and responding to the source.
All business logic for responding or reacting to DHCP messages lives here.
Handlers live in the `handler/` directory.

## Listener

Responsible for listening for UDP packets on the specified address and port.
A default listener can be used.

## Server

Responsible for filtering for DHCP packets received by the listener and calling the specified handler.

## Functional description

Server(listener, handler(backend))
