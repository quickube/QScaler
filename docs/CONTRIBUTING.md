## How To Contribute

We appreciate contributions from the community to make QScaler even better. To contribute, follow the steps below:

1. Fork the QScaler repository to your GitHub account.
2. Clone the forked repository to your local machine:
```bash
git clone https://github.com/your-username/QScaler.git
```
3. Create a new branch to work on your feature or bug fix:
```bash
git checkout -b my-feature
```
4. Make your changes, following the coding guidelines outlined in this document.
5. Commit your changes with clear and descriptive commit messages and sign it:
```bash
git commit -s -m "fix: Add new feature"
```
* please make sure you commit as described in [conventional commit](https://www.conventionalcommits.org/en/v1.0.0/)
6. Push your changes to your forked repository:
```bash
git push origin my-feature
```
7. Open a [pull request](#pull-requests) against the main branch of the original QScaler repository.

## Pull Requests

We welcome and appreciate contributions from the community. If you have developed a new feature, improvement, or bug fix for QScaler, follow these steps to submit a pull request:

1. Make sure you have forked the QScaler repository and created a new branch for your changes. Checkout [How To Contribute](#How-to-contribute).
2. commit your changes and push them to your forked repository.
3. Go to the QScaler repository on GitHub.
4. Click on the "New Pull Request" button.
5. Select your branch and provide a [descriptive title](#pull-request-nameing) and detailed description of your changes.
6. If your pull request relates to an open issue, reference the issue in the description using the GitHub issue syntax (e.g., Fixes #123).
7. Submit the pull request, and our team will review your changes. We appreciate your patience during the review process and may provide feedback or request further modifications.

### Pull Request Naming

The name should follow conventional commit naming. 

## Coding Guidelines

To maintain a consistent codebase and ensure readability, we follow a set of coding guidelines in QScaler. Please adhere to the following guidelines when making changes:

* Follow the [Effective Go](https://go.dev/doc/effective_go) guide for Go code.
* Follow the [Folder convention](https://github.com/golang-standards/project-layout) guide for Go code.
* Write clear and concise comments to explain the code's functionality.
* Use meaningful variable and function names.
* Make sure your code is properly formatted and free of syntax errors.
* Run tests locally.
* Check that the feature documented.
* Add new packages only if necessary and already existing one, can't be used.
* Add tests for new features or modification.


## Local deployment

To make it easy to develop locally, please run the following

Prerequisites :
1. install helm
```bash
brew install helm
```
2. install kubectl
```bash
brew install kubectl
```
3. install docker

4. install kind
```bash
brew install kind
```

Deployment:
1. make sure docker are running. 
2. use `make deploy`. it will do the following:
     * deploy a local registry as container
     * deploy a kind cluster as container with configuration
     * deploy QScaler with the local helm chart