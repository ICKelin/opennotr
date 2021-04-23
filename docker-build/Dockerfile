FROM ickelin/resty-upstream:latest
COPY opennotrd /opt/
COPY start.sh /opt/ 
RUN chmod +x /opt/start.sh && chmod +x /opt/opennotrd
CMD /opt/start.sh
