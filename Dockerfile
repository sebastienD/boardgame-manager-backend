# Étape 1 : Construire le binaire Go
# Utiliser une image officielle de Go pour la construction
FROM golang:1.22-alpine AS build

# Installer les dépendances nécessaires
RUN apk --no-cache add git

# Définir le répertoire de travail à l'intérieur du conteneur
WORKDIR /app

# Copier les fichiers go.mod et go.sum et télécharger les dépendances
COPY go.mod go.sum ./
RUN go mod download

# Copier le reste du code source
COPY . .

# Construire le binaire de l'application
RUN env GOOS=linux GOARCH=amd64 go build -o myapp .

# Étape 2 : Créer une image minimale pour exécuter l'application
FROM alpine:latest

# Installer les certificats CA pour les connexions HTTPS
RUN apk --no-cache add ca-certificates

# Définir le répertoire de travail à l'intérieur du conteneur
WORKDIR /root/

# Copier le binaire depuis l'image de build
COPY --from=build /app/myapp .

# Définir le point d'entrée de l'application
ENTRYPOINT ["./myapp"]
