#!/bin/bash

servicePrefixes=("accountingservice" "adservice" "cartservice" "checkoutservice" "currencyservice" "emailservice" "featureflagservice" "frauddetectionservice" "paymentservice" "productcatalogservice" "quoteservice" "recommendationservice" "shippingservice")

# # Loop over service prefix
for s in "${servicePrefixes[@]}"; do
  echo "Deleting service $s"
  # Find service for prefix
  name=$(kubectl get pods | grep ^"my-otel-demo-$s" | awk '{print $1}')

  # Delete service
  kubectl delete pod "$name"

  sleep 10
done

