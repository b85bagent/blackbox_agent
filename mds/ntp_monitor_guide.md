專案現在的 NTP 整合方式是：
  target.yaml 仍用 scrape_configs / static_configs.targets 來放監控標的，但真正的 probe 參數是定義在 blackbox_exporter/blackbox.yaml 的 modules.<name>，而且 NTP 需要用自訂的 prober: ntp 加上 ntp: 區塊，不是
  upstream blackbox 原生支援的 schema。可參考 yaml/target_sample.yaml:123, blackbox.yaml and internal/blackboxadapter/types.go:19.
  另外，target 可以直接寫主機名/IP；若沒帶 port，NTP 預設用 UDP 123，也可寫成 host:port，見 internal/blackboxadapter/custom_ntp.go:151 與 internal/blackboxadapter/custom_ntp.go:275。

  你要的設定方式可以這樣寫。

  blackbox_exporter/blackbox.yaml

  modules:
    ntp:
      prober: ntp
      timeout: 5s
      ntp:
        preferred_ip_protocol: ip4
        ip_protocol_fallback: true
        protocol_version: 4
        measurement_duration: 30s
        high_drift_threshold: 10ms
        # source_ip_address: 10.10.1.5   # 需要綁來源 IP 才加

  yaml/target.yaml

  scrape_configs:
    - job_name: ntp_probe
      metrics_path: /probe/ntp
      params:
        module:
          - ntp
      scrape_interval: 30s
      static_configs:
        - labels:
            check: ntp
          targets:
            - 10.10.1.10
            - 10.10.1.11
            - ntp.company.local

