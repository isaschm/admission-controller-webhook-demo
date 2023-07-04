#!/bin/bash

servicePrefixes=("accountingservice" "adservice" "cartservice" "checkoutservice" "currencyservice" "emailservice" "featureflagservice" "frauddetectionservice" "paymentservice" "productcatalogservice" "quoteservice" "recommendationservice" "shippingservice")

end=$((SECONDS+600))

while [ $SECONDS -lt $end ]; do
  # Loop over service prefixes
  for s in "${servicePrefixes[@]}"; do
    echo "Deleting service $s"
    # Find service for prefix
    name=$(kubectl get pods | grep ^"my-otel-demo-$s" | awk '{print $1}')

    # Delete service
    kubectl delete pod "$name"
  done
done
