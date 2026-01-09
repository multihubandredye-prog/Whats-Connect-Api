export default {
  name: "ChatMessages",
  data() {
    return {
      jid: "",
      messages: [],
      loading: false,
      searchQuery: "",
      startTime: "",
      endTime: "",
      isFromMe: "",
      onlyMedia: false,
      currentPage: 1,
      pageSize: 20,
      totalMessages: 0,
      // Media download tracking
      downloadedMedia: {}, // messageId -> { file_path, media_type, file_size, status }
      downloadingMedia: new Set(), // Set of messageIds currently downloading
      mediaDownloadErrors: {}, // messageId -> error message
      maxConcurrentDownloads: 3,
      currentDownloads: 0,
    };
  },
  computed: {
    totalPages() {
      return Math.ceil(this.totalMessages / this.pageSize);
    },
    formattedJid() {
      return (
        this.jid.trim() + (this.jid.includes("@") ? "" : "@s.whatsapp.net")
      );
    },
  },
  methods: {
    isValidForm() {
      return this.jid.trim().length > 0;
    },
    openModal() {
      // Check if there's a pre-selected JID from chat list
      const selectedJid = localStorage.getItem("selectedChatJid");
      if (selectedJid) {
        this.jid = selectedJid;
        localStorage.removeItem("selectedChatJid"); // Clean up

        this.loadMessages();
      }

      $("#modalChatMessages")
        .modal({
          onShow: function () {
            // Initialize accordion after modal is shown
            setTimeout(() => {
              $("#modalChatMessages .ui.accordion").accordion();
            }, 100);
          },
        })
        .modal("show");
    },
    async loadMessages() {
      if (!this.isValidForm()) {
        showErrorInfo("Por favor, insira um JID válido");
        return;
      }

      this.loading = true;
      try {
        const params = new URLSearchParams({
          offset: (this.currentPage - 1) * this.pageSize,
          limit: this.pageSize,
        });

        if (this.searchQuery.trim()) {
          params.append("search", this.searchQuery);
        }

        if (this.startTime) {
          params.append("start_time", this.startTime);
        }

        if (this.endTime) {
          params.append("end_time", this.endTime);
        }

        if (this.isFromMe !== "") {
          params.append("is_from_me", this.isFromMe);
        }

        if (this.onlyMedia) {
          params.append("media_only", "true");
        }

        const response = await window.http.get(
          `/chat/${this.formattedJid}/messages?${params}`
        );
        this.messages = response.data.results?.data || [];
        this.totalMessages = response.data.results?.pagination?.total || 0;

        if (this.messages.length === 0) {
          showErrorInfo("Nenhuma mensagem encontrada para os critérios especificados");
        } else {
          // Auto-download media for loaded messages
          this.downloadAllMediaInMessages();
        }
      } catch (error) {
        showErrorInfo(
          error.response?.data?.message || "Falha ao carregar mensagens"
        );
      } finally {
        this.loading = false;
      }
    },
    searchMessages() {
      this.currentPage = 1;
      this.loadMessages();
    },
    nextPage() {
      if (this.currentPage < this.totalPages) {
        this.currentPage++;
        this.loadMessages();
      }
    },
    prevPage() {
      if (this.currentPage > 1) {
        this.currentPage--;
        this.loadMessages();
      }
    },
    handleReset() {
      this.jid = "";
      this.messages = [];
      this.searchQuery = "";
      this.startTime = "";
      this.endTime = "";
      this.isFromMe = "";
      this.onlyMedia = false;
      this.currentPage = 1;
      this.totalMessages = 0;
      // Clear media download state
      this.downloadedMedia = {};
      this.downloadingMedia.clear();
      this.mediaDownloadErrors = {};
      this.currentDownloads = 0;
    },
    formatTimestamp(timestamp) {
      if (!timestamp) return "N/D";
      return moment(timestamp).format("MMM DD, YYYY HH:mm:ss");
    },
    formatMessageType(message) {
      if (message.media_type) return message.media_type.toUpperCase();
      if (message.message_type) return message.message_type.toUpperCase();
      return "TEXTO";
    },
    formatSender(message) {
      if (message.is_from_me) return "Eu";
      return message.push_name || message.sender_jid || "Desconhecido";
    },
    getMessageContent(message) {
      if (message.content) return message.content;
      if (message.text) return message.text;
      if (message.caption) return message.caption;
      if (message.media_type) return `[${message.media_type.toUpperCase()}]`;
      return "[Sem conteúdo]";
    },
    getMediaDisplay(message) {
      if (!message.media_type || !message.url || !message.id) {
        return null;
      }

      const messageId = message.id;
      const downloadedInfo = this.downloadedMedia[messageId];
      const isDownloaded = this.isMediaDownloaded(messageId);
      const isDownloading = this.isMediaDownloading(messageId);
      const hasError = this.hasMediaDownloadError(messageId);

      // Show loading state
      if (isDownloading) {
        return {
          type: 'loading',
          content: `<div class="ui active mini inline loader"></div> Baixando ${message.media_type}...`
        };
      }

      // Show error state with retry option
      if (hasError) {
        return {
          type: 'error',
          content: `<div class="ui red message">
            <i class="exclamation triangle icon"></i>
            Falha ao baixar ${message.media_type}
            <span class="ui mini button" style="cursor: pointer; margin-left: 10px;" 
                  onclick="document.dispatchEvent(new CustomEvent('retryMediaDownload', {detail: '${messageId}'}))">
              <i class="redo icon"></i> Tentar Novamente
            </span>
          </div>`
        };
      }

      // Show downloaded media
      if (isDownloaded && downloadedInfo) {
        const filePath = downloadedInfo.file_path;
        const mediaType = downloadedInfo.media_type;
        const filename = downloadedInfo.filename;
        const fileSize = downloadedInfo.file_size;

        switch (mediaType.toLowerCase()) {
          case 'image':
            return {
              type: 'image',
              content: `<div class="ui fluid image">
                <img src="${filePath}" alt="${filename}" style="max-width: 300px; max-height: 300px; border-radius: 4px;" 
                     onerror="this.style.display='none'; this.nextElementSibling.style.display='block';">
                <div style="display: none;" class="ui placeholder segment">
                  <div class="ui icon header">
                    <i class="image outline icon"></i>
                    Imagem não disponível
                  </div>
                </div>
              </div>`
            };

          case 'video':
            return {
              type: 'video',
              content: `<div class="ui fluid">
                <video controls style="max-width: 300px; max-height: 300px; border-radius: 4px;" preload="metadata">
                  <source src="${filePath}" type="video/mp4">
                  <source src="${filePath}" type="video/webm">
                  <source src="${filePath}" type="video/ogg">
                  Seu navegador não suporta a tag de vídeo.
                </video>
              </div>`
            };

          case 'audio':
            return {
              type: 'audio',
              content: `<div class="ui fluid">
                <audio controls style="width: 100%; max-width: 300px;">
                  <source src="${filePath}" type="audio/mpeg">
                  <source src="${filePath}" type="audio/ogg">
                  <source src="${filePath}" type="audio/wav">
                  Seu navegador não suporta a tag de áudio.
                </audio>
              </div>`
            };

          case 'document':
            const sizeText = fileSize ? `(${Math.round(fileSize / 1024)} KB)` : '';
            return {
              type: 'document',
              content: `<div class="ui labeled button">
                <a href="${filePath}" download="${filename}" class="ui button">
                  <i class="download icon"></i>
                  ${filename} ${sizeText}
                </a>
                <div class="ui basic left pointing label">
                  Documento
                </div>
              </div>`
            };

          case 'sticker':
            return {
              type: 'sticker',
              content: `<div class="ui">
                <img src="${filePath}" alt="Sticker" style="max-width: 150px; max-height: 150px; border-radius: 4px;" 
                     onerror="this.style.display='none'; this.nextElementSibling.style.display='block';">
                <div style="display: none;" class="ui placeholder segment">
                  <div class="ui icon header">
                    <i class="smile outline icon"></i>
                    Sticker não disponível
                  </div>
                </div>
              </div>`
            };

          default:
            return {
              type: 'unknown',
              content: `<div class="ui message">
                <i class="file icon"></i>
                Tipo de mídia desconhecido: ${mediaType}
              </div>`
            };
        }
      }

      // Default: show media available label
      return {
        type: 'available',
        content: `<div class="ui tiny blue label">
          <i class="linkify icon"></i>
          ${message.media_type.toUpperCase()} Disponível
        </div>`
      };
    },
    getMessageStyle(message) {
      const baseStyle = {
        padding: "1em",
        margin: "0.5em 0",
      };

      if (message.is_from_me) {
        return {
          ...baseStyle,
          borderLeft: "4px solid #2185d0",
          backgroundColor: "#f8f9fa",
        };
      } else {
        return {
          ...baseStyle,
          borderLeft: "4px solid #767676",
        };
      }
    },
    // Media download methods
    isMediaDownloaded(messageId) {
      return this.downloadedMedia[messageId] && this.downloadedMedia[messageId].status === 'completed';
    },
    isMediaDownloading(messageId) {
      return this.downloadingMedia.has(messageId);
    },
    hasMediaDownloadError(messageId) {
      return !!this.mediaDownloadErrors[messageId];
    },
    async downloadMediaForMessage(message) {
      if (!message.media_type || !message.url || !message.id) {
        return;
      }

      const messageId = message.id;
      
      // Skip if already downloaded or downloading
      if (this.isMediaDownloaded(messageId) || this.isMediaDownloading(messageId)) {
        return;
      }

      // Check concurrent download limit
      if (this.currentDownloads >= this.maxConcurrentDownloads) {
        return;
      }

      try {
        this.downloadingMedia.add(messageId);
        this.currentDownloads++;
        
        // Clear any previous error
        if (this.mediaDownloadErrors[messageId]) {
          delete this.mediaDownloadErrors[messageId];
        }

        const response = await window.http.get(
          `/message/${messageId}/download?phone=${this.formattedJid}`
        );

        if (response.data && response.data.results) {
          this.downloadedMedia[messageId] = {
            file_path: response.data.results.file_path,
            media_type: response.data.results.media_type,
            file_size: response.data.results.file_size,
            filename: response.data.results.filename,
            status: 'completed'
          };
        }
      } catch (error) {
        console.error(`Falha ao baixar mídia para a mensagem ${messageId}:`, error);
        this.mediaDownloadErrors[messageId] = error.response?.data?.message || 'Falha no download';
      } finally {
        this.downloadingMedia.delete(messageId);
        this.currentDownloads--;
      }
    },
    async retryMediaDownload(messageId) {
      const message = this.messages.find(m => m.id === messageId);
      if (message) {
        // Clear the error first
        delete this.mediaDownloadErrors[messageId];
        await this.downloadMediaForMessage(message);
      }
    },
    async downloadAllMediaInMessages() {
      const mediaMessages = this.messages.filter(message =>
        message.media_type && message.url && message.id &&
        !this.isMediaDownloaded(message.id) && !this.isMediaDownloading(message.id)
      );

      if (mediaMessages.length === 0) {
        return;
      }

      // Download in batches to respect concurrency limit
      const downloadQueue = [...mediaMessages];

      const processQueue = async () => {
        while (downloadQueue.length > 0 && this.currentDownloads < this.maxConcurrentDownloads) {
          const message = downloadQueue.shift();
          if (message) {
            await this.downloadMediaForMessage(message);
            // Small delay to prevent overwhelming the server
            await new Promise(resolve => setTimeout(resolve, 100));
          }
        }

        // If there are still items in queue and we can download more, continue
        if (downloadQueue.length > 0 && this.currentDownloads < this.maxConcurrentDownloads) {
          setTimeout(processQueue, 500); // Wait a bit before checking again
        }
      };

      // Start processing
      processQueue();
    },
    backToChatList() {
      // Close current modal
      $('#modalChatMessages').modal('hide');

      // Open Chat List modal after a short delay
      setTimeout(() => {
        if (window.ChatListComponent && window.ChatListComponent.openModal) {
          window.ChatListComponent.openModal();
        } else {
          // Fallback: try to find and click the Chat List card
          const chatListCards = document.querySelectorAll('.card .header');
          for (let card of chatListCards) {
            if (card.textContent.includes('Chat List')) {
              card.click();
              break;
            }
          }
        }
      }, 200);
    },
  },
  mounted() {
    // Expose the openModal method globally for ChatList component to call
    window.ChatMessagesComponent = this;

    // Handle retry media download events
    this.handleRetryMediaDownload = (event) => {
      const messageId = event.detail;
      this.retryMediaDownload(messageId);
    };

    // Listen for retry media download events
    document.addEventListener('retryMediaDownload', this.handleRetryMediaDownload);
  },
  beforeUnmount() {
    // Clean up global reference
    if (window.ChatMessagesComponent === this) {
      delete window.ChatMessagesComponent;
    }

    // Clean up event listeners
    if (this.handleRetryMediaDownload) {
      document.removeEventListener('retryMediaDownload', this.handleRetryMediaDownload);
    }
  },
  template: `
    <div class="purple card" @click="openModal()" style="cursor: pointer">
        <div class="content">
            <a class="ui purple right ribbon label">Chat</a>
            <div class="header">Mensagens de Conversa</div>
            <div class="description">
                Ver mensagens de conversas específicas com filtragem avançada
            </div>
        </div>
    </div>
    
    <!--  Modal ChatMessages  -->
    <div class="ui large modal" id="modalChatMessages">
        <i class="close icon"></i>
        <div class="header">
            <i class="comment icon"></i>
            Mensagens de Conversa
        </div>
        <div class="content">
            <div class="ui form">
                <div class="field">
                    <label>JID da Conversa</label>
                    <input type="text" 
                           placeholder="Insira o número de telefone ou JID completo (ex: 1234567890 ou id-do-grupo@g.us)" 
                           v-model="jid">
                </div>
                
                <div class="ui accordion">
                    <div class="title">
                        <i class="dropdown icon"></i>
                        Filtros Avançados (Opcional)
                    </div>
                    <div class="content">
                        <div class="fields">
                            <div class="eight wide field">
                                <label>Buscar Conteúdo da Mensagem</label>
                                <input type="text" 
                                       placeholder="Buscar no texto da mensagem..." 
                                       v-model="searchQuery">
                            </div>
                            <div class="four wide field">
                                <label>Filtro de Remetente</label>
                                <select class="ui dropdown" v-model="isFromMe">
                                    <option value="">Todas as mensagens</option>
                                    <option value="true">Minhas mensagens</option>
                                    <option value="false">Mensagens deles</option>
                                </select>
                            </div>
                            <div class="four wide field">
                                <label>&nbsp;</label>
                                <div class="ui checkbox">
                                    <input type="checkbox" v-model="onlyMedia">
                                    <label>Apenas mídia</label>
                                </div>
                            </div>
                        </div>
                        
                        <div class="fields">
                            <div class="eight wide field">
                                <label>Data/Hora de Início</label>
                                <input type="datetime-local" v-model="startTime">
                            </div>
                            <div class="eight wide field">
                                <label>Data/Hora de Término</label>
                                <input type="datetime-local" v-model="endTime">
                            </div>
                        </div>
                    </div>
                </div>
            </div>
            
            <div class="ui divider"></div>
            
            <div class="actions">
                <button class="ui primary button" 
                        :class="{'disabled': !isValidForm() || loading}"
                        @click="loadMessages">
                    <i class="search icon"></i>
                    {{ loading ? 'Carregando...' : 'Carregar Mensagens' }}
                </button>
                <button class="ui button" @click="handleReset">
                    <i class="refresh icon"></i>
                    Redefinir
                </button>
            </div>
            
            <div v-if="loading" class="ui active centered inline loader"></div>
            
            <div v-else-if="messages.length === 0 && totalMessages === 0" class="ui placeholder segment">
                <div class="ui icon header">
                    <i class="comment outline icon"></i>
                    Nenhuma mensagem carregada
                </div>
                <p>Insira um JID e clique em "Carregar Mensagens" para ver o histórico da conversa</p>
            </div>
            
            <div v-else-if="messages.length > 0">
                <div style="padding-top: 1em; padding-bottom: 1em;">
                    <div class="ui info message">
                        <div class="header">
                            Mensagens de Conversa para {{ formattedJid }}
                        </div>
                        <p>Exibindo {{ messages.length }} de {{ totalMessages }} mensagens</p>
                    </div>
                </div>
                
                <div class="ui divided items" style="max-height: 400px; overflow-y: auto; overflow-x: hidden; -webkit-overflow-scrolling: touch; scrollbar-width: thin;">
                    <div v-for="message in messages" :key="message.id" 
                         class="item" 
                         :style="getMessageStyle(message)">
                        <div class="content">
                            <div class="header">
                                <div class="ui horizontal label" 
                                     :class="message.is_from_me ? 'blue' : 'grey'">
                                    {{ formatSender(message) }}
                                </div>
                                <div class="ui right floated horizontal label">
                                    {{ formatMessageType(message) }}
                                </div>
                            </div>
                            <div class="meta">
                                <span>{{ formatTimestamp(message.timestamp) }}</span>
                                <span v-if="message.id" class="right floated">
                                    ID: {{ message.id }}
                                </span>
                            </div>
                            <div class="description">
                                <p>{{ getMessageContent(message) }}</p>
                                <div v-if="message.media_type && message.url" class="media-container" style="margin-top: 0.5em;">
                                    <div v-if="getMediaDisplay(message)" v-html="getMediaDisplay(message).content"></div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
                
                <!-- Pagination -->
                <div class="ui pagination menu" v-if="totalPages > 1">
                    <a class="icon item" @click="prevPage" :class="{ disabled: currentPage === 1 }">
                        <i class="left chevron icon"></i>
                    </a>
                    <div class="item">
                        Página {{ currentPage }} de {{ totalPages }}
                    </div>
                    <a class="icon item" @click="nextPage" :class="{ disabled: currentPage === totalPages }">
                        <i class="right chevron icon"></i>
                    </a>
                </div>
            </div>
        </div>
        <div class="actions">
            <button class="ui button" @click="backToChatList">
                <i class="arrow left icon"></i>
                Voltar para a Lista de Conversas
            </button>
            <div class="ui approve button">Fechar</div>
        </div>
    </div>
    `,
};
