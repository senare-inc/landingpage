# landingpage


docker run -it --rm \
-p 0.0.0.0:8080:8080 \
-v $(pwd)/landing/cfg:/cfg:ro \
landing:latest

# To 'release' create a new tag for reference

Set the variable in your current shell session, and use it to tag and push

```bash
export VERSION="v0.1.5" &&  git tag -a "$VERSION" -m "Release $VERSION" && git push origin "$VERSION"
```

for more information ref [Semantic Versioning 2.0.0](https://semver.org/)