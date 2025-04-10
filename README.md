# chess

This project is a *web-based chess application*.
It allows two players to play chess against each other in *real-time*.
The core game logic uses a **Bitboard Chess Engine** for efficiency.
User interactions happen through a web interface (frontend), which communicates with the backend via API calls and WebSockets.
Game states and user data are stored in a **PostgreSQL database** and cached using **Redis**.


```mermaid
flowchart TD
    A0["Chess Engine (Bitboard Implementation)"]
    A1["API Routing & Controllers"]
    A2["Data Persistence (Repositories)"]
    A3["Real-time Communication (WebSocket Service)"]
    A4["Service Layer"]
    A5["Domain Objects (DAO/DTO)"]
    A6["Dependency Injection & Configuration"]
    A7["Frontend Interaction"]
    A7 -- "Sends HTTP requests" --> A1
    A7 -- "Connects via WebSocket" --> A3
    A1 -- "Delegates request handling" --> A4
    A1 -- "Handles WebSocket upgrades" --> A3
    A4 -- "Uses for game logic (API)" --> A0
    A4 -- "Uses for data storage (API)" --> A2
    A3 -- "Uses for game logic (WS)" --> A0
    A3 -- "Uses for data storage (WS)" --> A2
    A2 -- "Maps data to/from DAOs" --> A5
    A0 -- "Operates on GameState DAO" --> A5
    A6 -- "Injects dependencies" --> A4
    A6 -- "Provides DB/Cache config" --> A2
    A4 -- "Uses DTOs/DAOs" --> A5
    A3 -- "Uses DTOs for messages" --> A5
    A6 -- "Injects dependencies" --> A1
    A6 -- "Injects dependencies" --> A3
```

## Chapters

1. [Frontend Interaction](docs/01_frontend_interaction.md)
2. [Domain Objects (DAO/DTO)](docs/02_domain_objects__dao_dto_.md)
3. [API Routing & Controllers](docs/03_api_routing___controllers.md)
4. [Real-time Communication (WebSocket Service)](docs/04_real_time_communication__websocket_service_.md)
5. [Service Layer](docs/05_service_layer.md)
6. [Chess Engine (Bitboard Implementation)](docs/06_chess_engine__bitboard_implementation_.md)
7. [Data Persistence (Repositories)](docs/07_data_persistence__repositories_.md)
8. [Dependency Injection & Configuration](docs/08_dependency_injection___configuration.md)


---
