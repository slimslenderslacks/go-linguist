# Building

Create a multi-platform image for the go-linguist tool.

```sh
#docker:command=build
docker buildx build --builder hydrobuild \
                    --platform linux/amd64,linux/arm64 \
                    --tag vonwig/go-linguist:latest \
                    --file Dockerfile \
                    --push .
```

## Testing

Try out the container on the project_root in $PWD using this.

```sh
cd /Users/slim/docker/labs-make-runbook && docker run -it --rm --workdir /project -v $PWD:/project vonwig/go-linguist:latest -json
```

## Pulling

```sh
docker pull vonwig/go-linguist:latest
```
