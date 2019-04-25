# Havuz CLI
The CLI application to establish connections to the edge instances in backend.

## Configuration

### Environment Variables:
| Var | Description | Example | Required |
| :---: | :---: | :---: | :---: |
| LICENSE | The license code for user authentication to backend. | | Yes |
| ADDR | Optional address to make gateway listen at. | :8080 (default) | No |

## Run
```
docker run -e LICENSE=<LICENSE_HERE> -d -p 8080:8080 havuz/havuz
```
`-d` flag detaches the container from terminal.
`-p` flag binds port to container (`-p host:container`). `-P` can be given instead to listen on a random port on host system.

## Logs
Useful logs are often output to stdout and stderr. `docker logs <CONTAINER_ID>` is your friend.
