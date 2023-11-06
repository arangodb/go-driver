# ArangoDB Go Driver V2

Implementation of Driver V2 makes use of runtime JSON/VPACK serialization, reducing memory and CPU Driver usage.

#### The V2 implementation is not considered production-ready yet. Also, not all features are implemented. See the table below.


## Features

| Implemented | Test Coverage | Description                                |
|-------------|---------------|--------------------------------------------|
|  ✓          |  ✓            | HTTP JSON & VPACK Connection               |
|  -          |  -            | HTTP2 JSON & VPACK Connection              |
|  -          |  -            | VST Connection                             |
|  +          |  +            | Database API Implementation                |
|  +          |  +            | Collection API Implementation              |
|  ✓          |  ✓            | Collection Document Creation               |
|  +          |  +            | Collection Document Update                 |
|  ✓          |  ✓            | Collection Document Read                   |
|  ✓          |  ✓            | Collection Document Delete                 |
|  ✓          |  ✓            | Collection Index                           |
|  ✓          |  ✓            | Query Execution                            |
|  +          |  -            | Transaction Execution                      |
|  +          |  +            | ArangoDB Operations (Views, Users, Graphs) |
